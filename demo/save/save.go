// Copyright 2012 Aaron Jacobs. All Rights Reserved.
// Author: aaronjjacobs@gmail.com (Aaron Jacobs)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"code.google.com/p/go.crypto/pbkdf2"
	crypto_rand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/jacobsa/aws/s3"
	"github.com/jacobsa/aws/sdb"
	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/config"
	"github.com/jacobsa/comeback/crypto"
	"github.com/jacobsa/comeback/fs"
	s3_kv "github.com/jacobsa/comeback/kv/s3"
	"github.com/jacobsa/comeback/sys"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"
)

// Keep this consistent with the item name used in the backup package.
const markerItemName = "comeback_marker"
const saltAttributeName = "password_salt"

var g_configFile = flag.String("config", "", "Path to config file.")
var g_jobName = flag.String("job", "", "Job name within the config file.")

func randUint64(randSrc *rand.Rand) uint64

// Return the existing salt used by the domain, or nil if there is none.
func getExistingSalt(domain sdb.Domain) (salt []byte, err error) {
	// Call the domain.
	attrs, err := domain.GetAttributes(
		markerItemName,
		false, // No need to ask for a consistent read
		[]string{saltAttributeName},
	)

	if err != nil {
		err = fmt.Errorf("GetAttributes: %v", err)
		return
	}

	if len(attrs) == 0 {
		return
	}

	if attrs[0].Name != saltAttributeName {
		panic(fmt.Errorf("Unexpected attribute: %v", attrs[0]))
	}

	// Base64-decode the salt.
	salt, err = base64.StdEncoding.DecodeString(attrs[0].Value)
	if err != nil {
		err = fmt.Errorf("base64.DecodeString(%s): %v", attrs[0].Value, err)
		return
	}

	return
}

func generateAndSetSalt(domain sdb.Domain) (salt []byte, err error) {
	const saltLen = 8

	// Generate a salt.
	salt = make([]byte, saltLen)
	if _, err = io.ReadAtLeast(crypto_rand.Reader, salt, len(salt)); err != nil {
		err = fmt.Errorf("Reading random bytes: %v", err)
		return
	}

	// Base64-encode it.
	encoded := base64.StdEncoding.EncodeToString(salt)

	// Write it out, making sure nobody has beaten us to the punch.
	err = domain.PutAttributes(
		markerItemName,
		[]sdb.PutUpdate{
			sdb.PutUpdate{Name: saltAttributeName, Value: encoded},
		},
		&sdb.Precondition{Name: saltAttributeName, Value: nil},
	)

	if err != nil {
		err = fmt.Errorf("PutAttributes: %v", err)
		return
	}

	return
}

func main() {
	var err error
	flag.Parse()

	// Attempt to read the user's config data.
	if *g_configFile == "" {
		fmt.Println("You must set -config.")
		os.Exit(1)
	}

	configData, err := ioutil.ReadFile(*g_configFile)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		os.Exit(1)
	}

	// Parse the config file.
	cfg, err := config.Parse(configData)
	if err != nil {
		fmt.Println("Parsing config file:", err)
		os.Exit(1)
	}

	// Validate the config file.
	if err := config.Validate(cfg); err != nil {
		fmt.Printf("Config file invalid: %v", err)
		os.Exit(1)
	}

	// Look for the specified job.
	if *g_jobName == "" {
		fmt.Println("You must set -job.")
		os.Exit(1)
	}

	job, ok := cfg.Jobs[*g_jobName]
	if !ok {
		fmt.Println("Unknown job:", *g_jobName)
		os.Exit(1)
	}

	// Create a user registry.
	userRegistry, err := sys.NewUserRegistry()
	if err != nil {
		log.Fatalf("Creating user registry: %v", err)
	}

	// Create a group registry.
	groupRegistry, err := sys.NewGroupRegistry()
	if err != nil {
		log.Fatalf("Creating group registry: %v", err)
	}

	// Create a file system.
	fileSystem, err := fs.NewFileSystem(userRegistry, groupRegistry)
	if err != nil {
		log.Fatalf("Creating file system: %v", err)
	}

	// Open a connection to SimpleDB.
	db, err := sdb.NewSimpleDB(cfg.SdbRegion, cfg.AccessKey)
	if err != nil {
		log.Fatalf("Creating SimpleDB: %v", err)
	}

	// Open the appropriate domain.
	domain, err := db.OpenDomain(cfg.SdbDomain)
	if err != nil {
		log.Fatalf("OpenDomain: %v", err)
	}

	// Open a connection to S3.
	bucket, err := s3.OpenBucket(cfg.S3Bucket, cfg.S3Region, cfg.AccessKey)
	if err != nil {
		log.Fatalf("Creating bucket: %v", err)
	}

	// Read in a password.
	password := []byte(readPassword("Enter crypto password: "))
	if len(password) == 0 {
		log.Fatalf("You must enter a password.")
	}

	// Look for a salt that has already been used.
	salt, err := getExistingSalt(domain)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// If there is no existing salt, set one up.
	if salt == nil {
		salt, err = generateAndSetSalt(domain)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	// Derive a crypto key from the password using PBKDF2, recommended for use by
	// NIST Special Publication 800-132. The latter says that PBKDF2 is approved
	// for use with HMAC and any approved hash function. Special Publication
	// 800-107 lists SHA-256 as an approved hash function.
	const pbkdf2Iters = 4096
	const keyLen = 32  // Minimum key length for AES-SIV
	cryptoKey := pbkdf2.Key(password, salt, pbkdf2Iters, keyLen, sha256.New)

	// Create the crypter.
	crypter, err := crypto.NewCrypter(cryptoKey)
	if err != nil {
		log.Fatalf("Creating crypter: %v", err)
	}

	// Create the backup registry.
	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))
	registry, err := backup.NewRegistry(domain, crypter, randSrc)
	if err != nil {
		log.Fatalf("Creating registry: %v", err)
	}

	// Create the kv store.
	kvStore, err := s3_kv.NewS3KvStore(bucket)
	if err != nil {
		log.Fatalf("Creating kv store: %v", err)
	}

	// Create the blob store.
	blobStore := blob.NewKvBasedBlobStore(kvStore)
	blobStore = blob.NewCheckingStore(blobStore)
	blobStore = blob.NewEncryptingStore(crypter, blobStore)

	// Create the file saver.
	fileSaver, err := backup.NewFileSaver(blobStore, 1<<24)
	if err != nil {
		log.Fatalf("Creating file saver: %v", err)
	}

	// Create a directory saver.
	dirSaver, err := backup.NewDirectorySaver(
		blobStore,
		fileSystem,
		fileSaver)

	if err != nil {
		log.Fatalf("Creating directory saver: %v", err)
	}

	// Choose a start time for the job.
	startTime := time.Now()

	// Run the job.
	score, err := dirSaver.Save(job.BasePath, "", job.Excludes)
	if err != nil {
		log.Fatalf("Saving: %v", err)
	}

	// Register the successful backup.
	completedJob := backup.CompletedJob{
		Id:        randUint64(randSrc),
		Name:      *g_jobName,
		StartTime: startTime,
		Score:     score,
	}

	if err = registry.RecordBackup(completedJob); err != nil {
		log.Fatalf("Recoding to registry: %v", err)
	}

	fmt.Printf("Successfully backed up. ID: %16x\n", completedJob.Id)
}
