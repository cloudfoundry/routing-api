# CF Routing API

## Installing this repo

To clone this repo you will need to have godeps installed. You can install godeps
by running the command `go get github.com/tools/godep`.

To then install this repo you can run the following commands.

```sh
go get github.com/pivotal-cf-experimental/routing-api
cd $GOPATH/src/github.com/pivotal-cf-experimental/routing-api
godep restore
```

## Development

To run the tests you need a running etcd cluster on version 0.4.6. To get that do:

```sh
go get github.com/coreos/etcd
cd $GOPATH/src/github.com/coreos/etcd
git fetch --tags
git checkout v0.4.6
go install .
```

Once installed, you can run etcd with the command `etcd` and you should see the
following output:
```
   | Using the directory majestic.etcd as the etcd curation directory because a directory was not specified.
   | majestic is starting a new cluster
   | etcd server [name majestic, listen on :4001, advertised url http://127.0.0.1:4001]   <-- default location of the etcd server
   | peer server [name majestic, listen on :7001, advertised url http://127.0.0.1:7001]
```

Note that this will run an etcd server and create a new directory at that location 
where it stores all of the records. This directory can be removed afterwards, or 
you can simply run etcd in a temporary directory.

## Usage

To run the API server you need to provide all the urls for the etcd cluster, plus some optional flags.

Example 1:

```sh
routing-api -port 3000 -maxTTL 60 http://etcd.127.0.0.1.xip.io:4001
```

Where `http://etcd.127.0.0.1.xip.io:4001` is the single etcd member.

Example 2:

```sh
routing-api http://etcd.127.0.0.1.xip.io:4001 http://etcd.127.0.0.1.xip.io:4002
```

Where `http://etcd.127.0.0.1.xip.io:4001` is one member of the cluster and `http://etcd.127.0.0.1.xip.io:4002` is another.

Note that flags have to come before the etcd addresses.
