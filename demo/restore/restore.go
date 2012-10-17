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
	"crypto/sha256"
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
	"github.com/jacobsa/comeback/registry"
	"github.com/jacobsa/comeback/sys"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

var g_configFile = flag.String("config", "", "Path to config file.")
var g_jobIdStr = flag.String("job_id", "", "The job ID to restore.")
var g_target = flag.String("target", "", "The target directory.")

func main() {
	var err error
	flag.Parse()

	// Parse the job ID.
	if len(*g_jobIdStr) != 16 {
		fmt.Println("You must set -job_id.")
		os.Exit(1)
	}

	jobId, err := strconv.ParseUint(*g_jobIdStr, 16, 64)
	if err != nil {
		fmt.Println("Invalid job ID:", *g_jobIdStr)
		os.Exit(1)
	}

	// Check the target.
	if *g_target == "" {
		fmt.Println("You must set -target.")
		os.Exit(1)
	}

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

	// Read in the AWS access key secret.
	cfg.AccessKey.Secret = readPassword("Enter AWS access key secret: ")
	if len(cfg.AccessKey.Secret) == 0 {
		log.Fatalf("You must enter an access key secret.\n")
	}

	// Validate the config file.
	if err := config.Validate(cfg); err != nil {
		fmt.Printf("Config file invalid: %v\n", err)
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

	// Read in the password.
	password := readPassword("Enter crypto password: ")
	if len(password) == 0 {
		log.Fatalf("You must enter a password.")
	}

	// Derive a crypto key from the password using PBKDF2, recommended for use by
	// NIST Special Publication 800-132. The latter says that PBKDF2 is approved
	// for use with HMAC and any approved hash function. Special Publication
	// 800-107 lists SHA-256 as an approved hash function.
	const pbkdf2Iters = 4096
	const keyLen = 32 // Minimum key length for AES-SIV
	keyDeriver := crypto.NewPbkdf2KeyDeriver(pbkdf2Iters, keyLen, sha256.New)

	// Create the backup registry.
	reg, crypter, err := registry.NewRegistry(domain, password, keyDeriver)
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

	// Create file restorer.
	fileRestorer, err := backup.NewFileRestorer(
		blobStore,
		fileSystem,
	)

	if err != nil {
		log.Fatalln("NewFileRestorer:", err)
	}

	// Create directory restorer.
	dirRestorer, err := backup.NewDirectoryRestorer(
		blobStore,
		fileSystem,
		fileRestorer,
	)

	if err != nil {
		log.Fatalln("NewDirectoryRestorer:", err)
	}

	// Find the requested job.
	job, err := reg.FindBackup(jobId)
	if err != nil {
		log.Fatalln("FindBackup:", err)
	}

	// Make sure the target doesn't exist.
	err = os.RemoveAll(*g_target)
	if err != nil {
		log.Fatalf("RemoveAll: %v", err)
	}

	// Create the target.
	err = os.Mkdir(*g_target, 0755)
	if err != nil {
		log.Fatalf("Mkdir: %v", err)
	}

	// Attempt a restore.
	err = dirRestorer.RestoreDirectory(
		job.Score,
		*g_target,
		"",
	)

	if err != nil {
		log.Fatalf("Restoring: %v", err)
	}
}
