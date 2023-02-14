
# gRPC demo cache storage server

This application demonstrates a mini version of memcached implemented
entirely in Golang, using gRPC to communicate between the client and 
the server.

The client is implemented as a console app that connects to the server,
obtains a "private" ID, and lets the user type commands to set and retrieve
values from the cache server.

## Compiling

To compile and run the server side, type

go run project\cmd\cacheserver\cacheserver.go

Optional command line parameter is --port, which indicates the TCP
port to bind to (on the localhost interface)

To compile and run the client side, type

go run project\cmd\cacheclient\cacheclient.go

Optional command line parameter is --addr, which should be in the
syntax host:port - this is where the server will be contacted

## Client commands

### set

set owner:service:name=value[,expiry]

This will set a cache entry in the server, with an optional expiry
value expressed as number of seconds, which can later be retrieved
via get or subscribe

### get

get owner:service:name

This will retrieve a cache entry from the server, in case one is
present

### subscribe

subscribe owner:service:name

This will subscribe for a cache entry on the server. If this, or
another, client sets the cache entry, the server will push a notification
via a gRPC stream to this client, displaying the new cache item
value on the screen
