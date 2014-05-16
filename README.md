Clients
=======

PHP: http://code.svn.wordpress.org/lockd/lockd-client.php
Python: https://gist.github.com/mdawaffe/e53c86e5163b48d5fe3a
Go: https://github.com/apokalyptik/glockc

Quickest Start
==============

Use one of the precompiled glockd binaries located in the subdirectories under "builds"

Docker Quick Start
===================
docker build -t glockd github.com/apokalyptik/glockd.git
docker run -p 9999:9999 -p 9998:9998 glockd -dump=false -registry=false -verbose=true

Quick Start
============

Option 1 (compile on demand)
----------------------------
cd glockd
go run ./*.go -pidfile my.pid -port 9999 -ws 9998

Option 2 (compile and then run)
-------------------------------
cd glockd
go build
./glockd --pidfile my.pid -port 9999 -ws 9998

Quick Start Testing
===================

cd tester
go run test.go --host 127.0.0.1:9999

Connecting to a glockd server
=============================

TCP/IP
------
If TCPIP has not been disabled (by passing -port 0 as a command line parameter) then you may simply
telnet to the port number that glockd is listening on (9999 by default.)  You can open a TCPIP socket
in any programming language this way (fsockopen in PHP for example.)  There is no handshake, banner, or
negotiation that takes place. Once connected it is ready for commands to be issued on the connection.

Client Implimentations:
	PHP		http://code.svn.wordpress.org/lockd/lockd-client.php

Websockets
----------
If websockets have not been disabled (by passing -ws 0 as a command line parameter) then you may simply
connect to it as you normally would on "ws://%s:%d/", host, port.  Glockd listens for websocket 
connections on port 9998 by default. The api is not changed at all for websockets.

Unix Sockets
------------
If a path to a local unix socket has been specified (via the -unix parameter) then you may connect to it
how you would any AF_UNIX socket per your programming language (example below.)  You may then read/write
commands as you would a TCP/IP socket connection.  This obviously only works when connecting to glockd
from the same machine since a shared filesystem which supports unix sockets is required.

Example connecting to glockd via unix sockets (python)

	import socket
	s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
	s.connect("/var/run/glockd/socket")
	s.sendall("g foo\n")
	print s.recv(4096)

Lock Types
==========

Exclusive Locks
---------------
Exclusive locks are... exclusive. They can only be held by one connection at a time.

Upon disconnection of a client all of that clients exclusive locks are considered to be "orphaned".
All orphaned locks are automatically released.  The intended purpose of this functionality is to help
avoid the complicated gymnastics generally used in distributed locking (timeouts, heartbeating, etc)
where a single process can maintain a lock simply by its continued presence, and can release its lock
by its absence. A side effect of this methodology is that stale locks simply cannot exist in this
environment.  If the connection goes away then its locks are released. Any lock still extant, therefor,
is still validly held by a process that is still literally running somewhere.

Shared Locks
------------
Shared locks are... not exclusive.  They can be obtained by any number of clients at the same time.

One interesting feature of shared locks is that they are counted. That is if 4 people have a lock, 
and another goes to lock the same thing then when it does it will be told that it is the 5th client
to obtain that lock.  This makes shared locks good for things like rate limiting, throttling, etc,
where the client can have logic built in in which after 5 active locks are obtained it waits, defers,
or otherwise avoids doing work for which the shared lock was wanted.

Upon disconnection of a client all of that clients shared locks are considered to be "orphaned".
All orphaned locks are automatically released. This behavior works just like the exclusive lock
orphaning feature.  Counts on shared locks are appropriately updated when locks are orphaned.

Exclusive Locks API
===================

Generally speaking most commands for exclusive locks return a response thusly: "%d %s".  The integer
portion of the response is meant for programatic interpretation, and represent success (1) or 
failure (0), or represent taken (1) or available (0).

Get a lock: "g %s\n", lockname
------------------------------
In the following example "foo" is available, but "bar" is already locked by another client

> g foo
< 1 Lock Get Success: foo
> g bar
< 0 Lock Get Failure: bar

Release a lock: "r %s\n", lockname
----------------------------------
> g foo
< 1 Lock Get Success: foo
> r foo
< 1 Lock Release Success: foo
> r bar
< 0 Lock Release Failure: bar

Inspect a lock: "i %s\n", lockname
----------------------------------
> i foo
< 1 Lock Is Locked: foo
> i bar
< 0 Lock Not Locked: bar

Get a list of one or more locks and their locking connections: "d\n", or "d %s\n", lockname (only available when -dump is true)
-------------------------------------------------------------------------------------------------------------------------------
This is mainly useful for debugging

> d
< baz: 174.62.83.171:59060
< foo: 174.62.83.171:59056
< bar: 174.62.83.171:59060
< boo: 174.62.83.171:59060
> d foo
< foo: 174.62.83.171:59056

Get a printout of the lock data structure: "dump\n" (only available when -dump is true)
---------------------------------------------------------------------------------------
This is mainly useful for debugging

> dump
< map[boo:174.62.83.171:59060 baz:174.62.83.171:59060 foo:174.62.83.171:59056 bar:174.62.83.171:59060]

Shared Locks API
================

Generally speaking most commands for shared locks return a response thusly: "%d %s".  The integer
portion of the response is meant for programatic interpretation, and represent success %d >= 1 or 
failure %d == 0, or represent [lack of] concurrency %d >= 0

Get a shared lock: "sg %s\n"
----------------------------
client1> sg foo
client1< 1 Shared Lock Get Success: foo
client2> sg foo
client2< 2 Shared Lock Get Success: foo
client2> sg bar
client2< 1 Shared Lock Get Success: bar

Release a shared lock: "sr %s\n"
--------------------------------
client1> sg foo
client1< 1 Shared Lock Get Success: foo
client2> sg foo
client2< 2 Shared Lock Get Success: foo
client3> si foo
client3< 2 Shared Lock Is Locked: foo
client1> sr foo
client1< 1 Shared Lock Release Success: foo
client3> si foo
client3< 1 Shared Lock Is Locked: foo
client2> sr foo
client2< 1 Shared Lock Release Success: foo
client3> si foo
client3< 0 Shared Lock Not Locked: foo

Inspect a shared lock: "si %s\n"
--------------------------------
client1> si foo
client1< 0 Shared Lock Not Locked: foo
client1> sg foo
client1< 1 Shared Lock Get Success: foo
client2> si foo
client2< 1 Shared Lock Is Locked: foo
client2> sg foo
client2< 2 Shared Lock Get Success: foo
client1> si foo
client1< 2 Shared Lock Get Success: foo

Get a list of one or more locks and their locking connections: "sd\n", or "sd %s\n", lockname (only available when -dump is true)
---------------------------------------------------------------------------------------------------------------------------------
> sd
< blah: 174.62.83.171:59615
< bar: 174.62.83.171:59615
< foo: 174.62.83.171:59615
< foo: 174.62.83.171:59614
< baz: 174.62.83.171:59615
> sd foo
< foo: 174.62.83.171:59615
< foo: 174.62.83.171:59614

Get a printout of the lock data structure: "dump shared\n"
----------------------------------------------------------
> dump shared
< map[blah:[174.62.83.171:59615] bar:[174.62.83.171:59615] foo:[174.62.83.171:59615 174.62.83.171:59614] baz:[174.62.83.171:59615]]

Registry API
============

Return the name of your connection (even when -registry is not true)
--------------------------------------------------------------------
This command always returns two values. First the default connection name 
(which is what would be used in the output of the dump( shared)? commands) 
and the second is the registered name of the connection (which would be 
used in the output of the s?d commands and defaults to the first parameter 
if the iam command was not used to register a name for the current session)

< me
> 1 127.0.0.1:57871 127.0.0.1:57871
< iam foo
> 1 ok
< me
> 1 127.0.0.1:57871 foo

Set the name for your connection (only available when -registry is true)
------------------------------------------------------------------------
> g lock1
< 1 Got Lock
> d lock1
< lock1: 127.0.0.1:60882
> iam foo
< 1 ok
> d lock1
< lock1: foo
> iam
< 1 ok
> d lock1
< lock1: 127.0.0.1:60882

Find which clients have chosen to be a specific name (only available when -dump is true and -registry is true)
--------------------------------------------------------------------------------------------------------------
client1> who
client1< 
client1> iam me
client1< 1 ok
client2> iam someone_else
client2< 1 ok
client1> who
client1< 127.0.0.1:60882: me
client1< 127.0.0.1:60918: someone_else
client1> who someone_else
client1< 127.0.0.1:60918: someone_else

Stats API
=========

Get stats information: "q\n"
----------------------------
> q
< command_d: 4
< command_dump: 1
< command_g: 9
< command_i: 7
< command_q: 1
< command_r: 3
< command_sd: 1
< command_sg: 1
< command_si: 2
< command_sr: 1
< connections: 2
< invalid_commands: 23
< locks: 4
< orphans: 2
< shared_locks: 1
< shared_orphans: 1

Stats Response: "command_%s"
----------------------------
The number of times a particular command has been issued since the glockd has been running.
Zeroed on startup

Stats Response: "locks", "shared_locks"
---------------------------------------
The current active number of locked strings.  For shared locks this is the number of locked 
strings and NOT the number of clients with active locks

Stats Response: "orphans", "shared_orphans"
-------------------------------------------
Incrimented by one every time a lock is orphaned. If a client disconnects with 3 shared and 1
exclusive locks then the numbers are incrimented by 3 and 1 respecively. Zeroed on startup

Stats Response: "connections"
-----------------------------
The number of live connections to glockd. This number should always be 1 since you cannot
get these stats except by connecting.

Stats Response: "invalid_commands"
----------------------------------
The number of times something has been sent to glockd but that something was not a valid
command. Example: "stats\n" would incriment this counter by one since "stats" is not a valid
command.  Zeroed on startup

