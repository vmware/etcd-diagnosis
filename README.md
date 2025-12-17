# etcd-diagnosis

etcd-diagnosis is a low-level etcd diagnosis tool. See the detailed usage below.

```
$ ./etcd-diagnosis -h
A comprehensive etcd diagnosis tool

Usage:
  etcd-diagnosis [command]

Available Commands:
  bbolt            A simple command line tool for inspecting bbolt databases
  commit-index     commit-index reads the commit index from data directory.
  completion       Generate the autocompletion script for the specified shell
  consistent-index consistent-index reads consistent_index from the meta bucket in the db file.
  hash             hash computes the hash of db file.
  help             Help about any command
  iterate-bucket   iterate-bucket lists key-value pairs in reverse order.
  list-bucket      list-bucket lists all buckets.
  log              log dumps the log from data directory.
  report           report generates a diagnostic report for a running etcd cluster.
  version          Prints the version of etcd-diagnosis

Flags:
  -h, --help               help for etcd-diagnosis
      --timeout duration   time to wait to obtain a file lock on db file, 0 to block indefinitely (default 10s)

Use "etcd-diagnosis [command] --help" for more information about a command.
```

Note that `report` is the only online command that requires a running
etcd cluster. All other commands are offline and should be executed only
when all etcd instances are stopped. The tool is expected to be executed
on one of the nodes hosting etcd.

- Online commands
  - [report](#report)
- Offline commands
  - [list-bucket](#list-bucket)
  - [iterate-bucket](#iterate-bucket)
  - [hash](#hash)
  - [log](#log)
  - [consistent-index](#consistent-index)
  - [commit-index](#commit-index)
  - [bbolt](#bbolt)

## list-bucket
`list-bucket` command lists all top-level buckets in an etcd db file.
The parameter can be either the path to the etcd data directory or the
database file itself.

Usage,
```
$ ./etcd-diagnosis list-bucket -h
list-bucket lists all buckets.

Usage:
  etcd-diagnosis list-bucket [data dir or db file path] [flags]

Flags:
  -h, --help   help for list-bucket

Global Flags:
      --timeout duration   time to wait to obtain a file lock on db file, 0 to block indefinitely (default 10s)
```

Example,
```
$ ./etcd-diagnosis list-bucket ~/tmp/etcd/default.etcd/member/snap/db
alarm
auth
authRoles
authUsers
cluster
key
lease
members
members_removed
meta
```

## iterate-bucket
`iterate-bucket` command lists key-value pair of a given bucket in an
etcd db file in reverse order. The first parameter can be either the
path to the etcd data directory or the database file itself. The second
parameter is the bucket name.

Usage,
```
$ ./etcd-diagnosis iterate-bucket -h
iterate-bucket lists key-value pairs in reverse order.

Usage:
  etcd-diagnosis iterate-bucket [data dir or db file path] [bucket name] [flags]

Flags:
      --decode       true to decode Protocol Buffer encoded data
  -h, --help         help for iterate-bucket
      --limit uint   max number of key-value pairs to iterate (0 to iterate all)

Global Flags:
      --timeout duration   time to wait to obtain a file lock on db file, 0 to block indefinitely (default 10s)
```

Examples,
```
$ ./etcd-diagnosis iterate-bucket ~/tmp/etcd/default.etcd/ key --decode
rev={Revision:{Main:4 Sub:0} tombstone:false}, value=[key "k3" | val "v3" | created 4 | mod 4 | ver 1]
rev={Revision:{Main:3 Sub:0} tombstone:false}, value=[key "k2" | val "v2" | created 3 | mod 3 | ver 1]
rev={Revision:{Main:2 Sub:0} tombstone:false}, value=[key "k1" | val "v1" | created 2 | mod 2 | ver 1]

$ ./etcd-diagnosis iterate-bucket ~/tmp/etcd/default.etcd/ meta --decode
key="term", value=2
key="consistent_index", value=7
key="confState", value="{\"voters\":[10276657743932975437],\"auto_leave\":false}"

$ ./etcd-diagnosis iterate-bucket ~/tmp/etcd/infra1.etcd/ members --decode
key="fd422379fda50e48", value="{\"id\":18249187646912138824,\"peerURLs\":[\"http://127.0.0.1:32380\"],\"name\":\"infra3\",\"clientURLs\":[\"http://127.0.0.1:32379\"]}"
key="91bc3c398fb3c146", value="{\"id\":10501334649042878790,\"peerURLs\":[\"http://127.0.0.1:22380\"],\"name\":\"infra2\",\"clientURLs\":[\"http://127.0.0.1:22379\"]}"
key="8211f1d0f64f3269", value="{\"id\":9372538179322589801,\"peerURLs\":[\"http://127.0.0.1:12380\"],\"name\":\"infra1\",\"clientURLs\":[\"http://127.0.0.1:2379\"]}"

#  What data is consuming most of the storage space?
$ ./etcd-diagnosis iterate-bucket ~/box/open_source/etcd/data/k8s_1.21.5/db key --decode | egrep -o '"/registry.*' | cut -d'|' -f1 | grep -v ^$ | awk -F '/'  '{ h[$3]++ } END {for (k in h) print h[k], k}' | sort -nr
722 leases
79 clusterroles
65 clusterrolebindings
58 nsx.vmware.com
57 secrets
51 serviceaccounts
42 apiregistration.k8s.io
36 masterleases
21 pods
14 configmaps
13 rolebindings
12 services
12 podsecuritypolicy
11 roles
11 flowschemas
11 apiextensions.k8s.io
9 replicasets
9 deployments
7 prioritylevelconfigurations
7 namespaces
6 minions
6 endpointslices
4 cns.vmware.com
3 daemonsets
3 csinodes
3 controllerrevisions
2 ranges
2 priorityclasses
1 vmware.com
1 validatingwebhookconfigurations
1 storageclasses
1 persistentvolumes
1 persistentvolumeclaims
1 jobs
1 csidrivers
```

## hash
`hash` command computes the hash of the db file. The parameter can be
either the path to the etcd data directory or the database file itself.

Usage,
```
$ ./etcd-diagnosis hash -h
hash computes the hash of db file.

Usage:
  etcd-diagnosis hash [data dir or db file path] [flags]

Flags:
  -h, --help   help for hash

Global Flags:
      --timeout duration   time to wait to obtain a file lock on db file, 0 to block indefinitely (default 10s)
```

Example,
```
$ ./etcd-diagnosis hash ~/tmp/etcd/default.etcd/
db path: /Users/wachao/tmp/etcd/default.etcd/member/snap/db
Hash: 1099832664
```

## log
`log` dumps the log from data directory. The parameter must be the
path to the etcd data directory.

Usage,
```
$ ./etcd-diagnosis log -h
log dumps the log from data directory.

Usage:
  etcd-diagnosis log [data dir] [flags]

Flags:
      --end-index uint      the index to stop dumping (exclusive) (default 18446744073709551615)
      --entry-type string   If set, filters output by entry type. Must be one or more than one of:
                            ConfigChange, Normal, Request, InternalRaftRequest,
                            IRRRange, IRRPut, IRRDeleteRange, IRRTxn,
                            IRRCompaction, IRRLeaseGrant, IRRLeaseRevoke, IRRLeaseCheckpoint (default "Normal,ConfigChange")
  -h, --help                help for log
      --raw                 read the logs in the low-level form
      --start-index uint    the index to start dumping (inclusive). If unspecified, dumps from the index of the last snapshot
      --start-snap string   the base name of snapshot file to start dumping
      --wal-dir string      if set, dumps WAL from the informed path, rather than following the standard 'data_dir/member/wal/' location

Global Flags:
      --timeout duration   time to wait to obtain a file lock on db file, 0 to block indefinitely (default 10s)
```

Example,
```
$ ./etcd-diagnosis log ~/tmp/etcd/infra1.etcd/
Snapshot:
empty
Start dumping log entries from snapshot.
WAL metadata:
nodeID=8211f1d0f64f3269 clusterID=ef37ad9dc622a7c4 term=2 commitIndex=22 vote=91bc3c398fb3c146
WAL entries: 23
lastIndex=23
term	     index	type	data
   1	         1	conf	method=ConfChangeAddNode id=8211f1d0f64f3269
   1	         2	conf	method=ConfChangeAddNode id=91bc3c398fb3c146
   1	         3	conf	method=ConfChangeAddNode id=fd422379fda50e48
   2	         4	norm
   2	         5	norm	method=PUT path="/0/members/91bc3c398fb3c146/attributes" val="{\"name\":\"infra2\",\"clientURLs\":[\"http://127.0.0.1:22379\"]}"
   2	         6	norm	method=PUT path="/0/members/8211f1d0f64f3269/attributes" val="{\"name\":\"infra1\",\"clientURLs\":[\"http://127.0.0.1:2379\"]}"
   2	         7	norm	method=PUT path="/0/members/fd422379fda50e48/attributes" val="{\"name\":\"infra3\",\"clientURLs\":[\"http://127.0.0.1:32379\"]}"
   2	         8	norm	method=PUT path="/0/version" val="3.5.0"
   2	         9	norm	header:<ID:3632602622773439492 > put:<key:"k1" value:"v1" >
   2	        10	norm	header:<ID:3632602622773439493 > put:<key:"k1" value:"v1" >
   2	        11	norm	header:<ID:3632602622773439494 > put:<key:"k1" value:"v1" >
   2	        12	norm	header:<ID:3632602622773439495 > put:<key:"k1" value:"v1" >
   2	        13	norm	header:<ID:3632602622773439496 > put:<key:"k1" value:"v1" >
   2	        14	norm	header:<ID:3632602622773439497 > put:<key:"k1" value:"v1" >
   2	        15	norm	header:<ID:3632602622773439498 > put:<key:"k1" value:"v1" >
   2	        16	norm	header:<ID:3632602622773439499 > put:<key:"k1" value:"v1" >
   2	        17	norm	header:<ID:3632602622773439500 > put:<key:"k1" value:"v1" >
   2	        18	norm	header:<ID:3632602622773439501 > put:<key:"k1" value:"v1" >
   2	        19	norm	header:<ID:3632602622773439502 > put:<key:"k1" value:"v1" >
   2	        20	norm	header:<ID:13926986946012262662 > alarm:<>
   2	        21	norm	header:<ID:3632602622773439505 > alarm:<>
   2	        22	norm	header:<ID:1029240563176583429 > alarm:<>
   2	        23	norm	header:<ID:3632602622773439508 > compaction:<revision:12 physical:true >

Entry types (Normal,ConfigChange) count is : 23
```

## consistent-index
`consistent-index` reads the consistent_index from the meta bucket in
the db file. The parameter can be either the path to the etcd data
directory or the database file itself.

Usage,
```
$ ./etcd-diagnosis consistent-index -h
consistent-index reads consistent_index from the meta bucket in the db file.

Usage:
  etcd-diagnosis consistent-index [data dir or db file path] [flags]

Flags:
  -h, --help   help for consistent-index

Global Flags:
      --timeout duration   time to wait to obtain a file lock on db file, 0 to block indefinitely (default 10s)
```

Example,
```
$ ./etcd-diagnosis consistent-index ~/tmp/etcd/infra1.etcd/
23
```

## commit-index
`commit-index` reads the commit index from data directory.
The parameter must be the path to the etcd data directory.

Usage,
```
$ ./etcd-diagnosis commit-index -h
commit-index reads the commit index from data directory.

Usage:
  etcd-diagnosis commit-index [data dir] [flags]

Flags:
  -h, --help             help for commit-index
      --wal-dir string   path to the dedicated wal directory, defaults to '${data_dir}/member/wal/' if not set

Global Flags:
      --timeout duration   time to wait to obtain a file lock on db file, 0 to block indefinitely (default 10s)
```

Example,
```
$ ./etcd-diagnosis commit-index ~/tmp/etcd/infra1.etcd/
23
```

## report
`report` generates a diagnostic report for a running etcd cluster.
Note the `report` command requires the etcd cluster to be running
to generate diagnostic reports, unlike other commands that operate
on the data directory or database file and require the cluster to
be stopped.

It reuses most of the `etcdctl` global flags, offering a familiar
experience to etcdctl users. See detailed usage below,

```
$ ./etcd-diagnosis report -h
report generates a diagnostic report for a running etcd cluster.

Usage:
  etcd-diagnosis report [flags]

Flags:
      --cacert string                  verify certificates of TLS-enabled secure servers using this CA bundle
      --cert string                    identify secure client using this TLS certificate file
      --cluster                        use all endpoints from the cluster member list
      --command-timeout duration       command timeout (excluding dial timeout) (default 5s)
      --dial-timeout duration          dial timeout for client connections (default 2s)
  -d, --discovery-srv string           domain name to query for SRV records describing cluster endpoints
      --discovery-srv-name string      service name to query when using DNS discovery
      --endpoints strings              comma separated etcd endpoints (default [127.0.0.1:2379])
      --etcd-storage-quota-bytes int   etcd storage quota in bytes (the value passed to etcd instance by flag --quota-backend-bytes) (default 2147483648)
  -h, --help                           help for report
      --insecure-discovery             accept insecure SRV records describing cluster endpoints (default true)
      --insecure-skip-tls-verify       skip server certificate verification (CAUTION: this option should be enabled only for testing purposes)
      --insecure-transport             disable transport security for client connections (default true)
      --keepalive-time duration        keepalive time for client connections (default 2s)
      --keepalive-timeout duration     keepalive timeout for client connections (default 5s)
      --key string                     identify secure client using this TLS key file
      --output string                  file path to write the online diagnosis report (default "etcd_diagnosis_report.json")
      --password string                password for authentication (if this option is used, --user option shouldn't include password)
      --user string                    username[:password] for authentication (prompt if password is not supplied)

Global Flags:
      --timeout duration   time to wait to obtain a file lock on db file, 0 to block indefinitely (default 10s)
```

Example,
```
./etcd-diagnosis report --endpoints=https://10.200.6.179:2379,https://10.200.6.187:2379,https://10.200.6.188:2379
```

Note if the endpoints are HTTPS URLs, and you do not specify values for
`--cacert`, `--cert` and `key`, then the following default values are used,
```
--cacert=/etc/kubernetes/pki/etcd/ca.crt
--cert=/etc/kubernetes/pki/etcd/healthcheck-client.crt
--key=/etc/kubernetes/pki/etcd/healthcheck-client.key
```


Check the [example report](./commands/report/examples/etcd_diagnosis_report.json) generated by the command.

## bbolt
`bbolt` integrates all CLI commands implemented in [go.etcd.io/bbolt/cmd/bbolt/command](https://github.com/etcd-io/bbolt/tree/main/cmd/bbolt/command).

Usage,
```
$ ./etcd-diagnosis bbolt -h
A simple command line tool for inspecting bbolt databases

Usage:
  etcd-diagnosis bbolt [command]

Available Commands:
  bench       run synthetic benchmark against bbolt
  buckets     print a list of buckets in bbolt database
  check       verify integrity of bbolt database data
  compact     creates a compacted copy of the database from source path to the destination path, preserving the original.
  dump        prints a hexadecimal dump of one or more pages of bbolt database.
  get         get the value of a key from a (sub)bucket in a bbolt database
  info        prints basic information about the bbolt database.
  inspect     inspect the structure of the database
  keys        print a list of keys in the given (sub)bucket in bbolt database
  page        page prints one or more pages in human readable format.
  page-item   print a page item key and value in a bbolt database
  pages       print a list of pages in bbolt database
  stats       print stats of bbolt database
  surgery     surgery related commands
  version     print the current version of bbolt

Flags:
  -h, --help      help for bbolt
  -v, --version   version for bbolt

Global Flags:
      --timeout duration   time to wait to obtain a file lock on db file, 0 to block indefinitely (default 10s)

Use "etcd-diagnosis bbolt [command] --help" for more information about a command.
```

For more detailed document, refer to [here](https://github.com/etcd-io/bbolt/tree/main/cmd/bbolt).
