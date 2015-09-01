The following things may be surprising:

*   Hard links between files in a backup are not preserved—if you restore the
    backup or mount it using `comeback mount`, files that previously shared the
    same inode will appear as independent files that happen to have the same
    content.

*   Although ownership information is recorded when saving a backup, it is not
    currently restored when restoring a backup and is not exposed through the
    file system when using `comeback mount`. In both cases, the UID and GID
    seen is that of the `comeback` process.

    The ownership information is recorded
    [here][save.dependencyResolver.FindDependencies]. The relevant restore code
    is [here][restore.newVisitor], and fuse file system code is
    [here][comebackfs.NewFileSystem].


[save.dependencyResolver.FindDependencies]: https://github.com/jacobsa/comeback/blob/2ead6ca/internal/save/dependency_resolver.go#L107
[restore.newVisitor]: https://github.com/jacobsa/comeback/blob/016abc4/internal/restore/visitor.go#L42
[comebackfs.NewFileSystem]: https://github.com/jacobsa/comeback/blob/2ead6ca/internal/comebackfs/fs.go#L36-L37
