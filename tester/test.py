#!/usr/bin/env python
import socket
import random
import string
import re
from optparse import OptionParser

# pip install websocket-client
from websocket import create_connection

parser = OptionParser()
parser.add_option("-u", "--unix", action="store", type="string", dest="unix", default="/tmp/glockd.sock", help="path to the unix socket")
parser.add_option("-t", "--tcp", action="store", type="string", dest="tcp", default="127.0.0.1:9999", help="address:port to the tcp socket")
parser.add_option("-w", "--ws", action="store", type="string", dest="ws", default="ws://127.0.0.1:9998/", help="url for the websocket listener")
(options, args) = parser.parse_args()

class gsock:
	def cmd(self, data):
		self.socket.sendall("%s\n" %data)
		(v, s) = self.socket.recv(1024000).strip().split(" ", 1)
		v = int(v)
		return (v, s)

	def raw(self, data):
		self.socket.sendall("%s\n" %data)
		return self.socket.recv(1024000).strip()

	def close(self):
		self.socket.close()

class gunix(gsock):
    def __init__(self, path):
        self.socket = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.socket.connect(path)

class gtcp(gsock):
	def __init__(self, address):
		self.socket = socket.socket(socket.AF_INET)
		(h, p) = address.split(":")
		p = int(p)
		self.socket.connect((h, p))

class gws(gsock):
	def __init__(self, address):
		self.socket = create_connection(address)

	def cmd(self, data):
		self.socket.send(data)
		(v, s) = self.socket.recv().strip().split(" ", 1)
		v = int(v)
		return (v, s)
	
	def raw(self, data):
		self.socket.send("%s" %data)
		return self.socket.recv().strip()

	def close(self):
		self.socket.close()

def test_registry(one, two):
	(i1, v1) = one.cmd( 'me' )
	(i2, v2) = two.cmd( 'me' )
	prefix = ok if v1 != v2 else no
	print prefix + "me\t\tclient1 and client2 have unique default identifiers"

	(i, v) = one.cmd( 'iam client1' )
	prefix = ok if i == 1 else no
	print prefix + "iam\t\tclient1 changed its name"

	(i, v) = one.cmd( 'me' )
	v1 = v1.split(" ", 1)
	v1[1] = "client1"
	v1 = " ".join(v1)
	prefix = ok if v == v1 else no
	print prefix + "me\t\tclient1 now shows proper new name via the me command"

def test_exclusive(one, two):
	(i, v) = one.cmd( 'i ' + random_lock_string )
	prefix = ok if i == 0 else no
	print prefix + "i\t\texclusive lock should not yet be held"

	(i, v) = one.cmd( 'g ' + random_lock_string )
	prefix = ok if i == 1 else no
	print prefix + "g\t\tfirst client should get exclusive lock"
	
	(i, v) = two.cmd( 'i ' + random_lock_string )
	prefix = ok if i == 1 else no
	print prefix + "i\t\texclusive lock should now be held"

	(i, v) = one.cmd( 'g ' + random_lock_string )
	prefix = ok if i == 1 else no
	print prefix + "g\t\tfirst client should get exclusive lock again if rerequested"

	(i, v) = two.cmd( 'g ' + random_lock_string )
	prefix = ok if i == 0 else no
	print prefix + "g\t\tsecond client should not get exclusive lock obtained by first client"

	(i, v) = two.cmd( 'r ' + random_lock_string )
	prefix = ok if i == 0 else no
	print prefix + "r\t\tsecond client should not be able to release an exclusive lock that it does not have"

	(i, v) = one.cmd( 'r ' + random_lock_string )
	refix = ok if i == 1 else no
	print prefix + "r\t\tfirst client should be able to release its exclusive lock"

	(i, v) = two.cmd( 'g ' + random_lock_string )
	prefix = ok if i == 1 else no
	print prefix + "g\t\tsecond client should be able to get the recently released exclusive lock"

	(i, v) = one.cmd( 'g ' + random_lock_string )
	prefix = ok if i == 0 else no
	print prefix + "g\t\tfirst client should not get exclusive lock obtained by second client"

def test_shared(one, two):
	(i, v) = one.cmd( 'si ' + random_lock_string )
	prefix = ok if i == 0 else no
	print prefix + "si\t\tshared lock should not be held"

	(i, v) = one.cmd( 'sr ' + random_lock_string )
	prefix = ok if i == 0 else no
	print prefix + "sr\t\tfirst client should not be able to release a shared lock that it has not obtained"

	(i, v) = one.cmd( 'sg ' + random_lock_string )
	prefix = ok if i == 1 else no
	print prefix + "sg\t\tfirst client should be able to get shared lock and see that it is the first client to do so"

	(i, v) = one.cmd( 'sg ' + random_lock_string )
	prefix = ok if i == 1 else no
	print prefix + "sg\t\tfirst client should be able to get shared lock again but not increment the counter"

	(i, v) = two.cmd( 'si ' + random_lock_string )
	prefix = ok if i == 1 else no
	print prefix + "si\t\tsecond client should now see the lock held (rval: 1)"

	(i, v) = two.cmd( 'sg ' + random_lock_string )
	prefix = ok if i == 2 else no
	print prefix + "sg\t\tsecond client should also get shared lock and see that it is the second client to do so"

	(i, v) = two.cmd( 'sg ' + random_lock_string )
	prefix = ok if i == 2 else no
	print prefix + "sg\t\tsecond client should also get shared lock again but not increment the counter"

	(i, v) = one.cmd( 'si ' + random_lock_string )
	prefix = ok if i == 2 else no
	print prefix + "si\t\tfirst client should now see the lock held by two clients"

	(i, v) = one.cmd( 'sr ' + random_lock_string )
	prefix = ok if i == 1 else no
	print prefix + "sr\t\tfirst client should be able to release a shared lock that it has obtained"

	(i, v) = two.cmd( 'si ' + random_lock_string )
	prefix = ok if i == 1 else no
	print prefix + "si\t\tsecond client should now see the lock held by one client"

def test_orphan(one, two):
	(i, v) = two.cmd( 'i ' + random_lock_string )
	prefix = ok if i == 1 else no
	print prefix + "i\t\tsecond client should have the exclusive lock"
	
	(i, v) = two.cmd( 'si ' + random_lock_string )
	prefix = ok if i == 1 else no
	print prefix + "si\t\tsecond client should have the shared lock"

	two.close()
	print ok + "\t\tsecond client has disconnected"

	(i, v) = one.cmd( 'i ' + random_lock_string )
	prefix = ok if i == 0 else no
	print prefix + "i\t\tfirst client should now see the exclusive lock as unlocked"

	(i, v) = one.cmd( 'si ' + random_lock_string )
	prefix = ok if i == 0 else no
	print prefix + "i\t\tfirst client should now see the shared lock as unlocked"

	one.close()
	print ok + "\t\tfirst client has disconnected"

def test_introspection(one, two):
	(i, v) = one.cmd( 'iam client1' )
	(i, me) = one.cmd( 'me' )
	(i, v) = two.cmd( 'iam client2' )
	v = one.raw( 'who' )
	prefix = ok if v.find('client1') > 0 and v.find('client2') > 0 else no
	print prefix + "who\t\tfound client1 and client2"

	v = two.raw( 'who client1' )
	prefix = ok if v == me.replace(" ", ": ") else no
	print prefix + "who\t\tresolved client1 to first connection via second connection who command"
	
	v = one.raw( 'd ' + random_lock_string )
	prefix = ok if v == random_lock_string + ": client2" else no
	print prefix + "d\t\texclusive lock held by client2"

	v = one.raw( 'sd ' + random_lock_string )
	prefix = ok if v == random_lock_string + ": client2" else no
	print prefix + "sd\t\tshared lock held by client2"

	v = one.raw( 'dump' )
	prefix = ok if v[0:4] == "map[" and v[-1:] == "]" else no
	print prefix + "dump\t\tdump looks like a GO map"

	v = one.raw( 'dump shared' )
	prefix = ok if v[0:4] == "map[" and v[-1:] == "]" else no
	print prefix + "dump shared\tdump looks like a GO map"

	v = one.raw( 'q' )
	prefix = ok if v.find('connections: ') > 0 and v.find('command_g') > 0 else no
	print prefix + "q\t\tstats output looks like stats"

def test(one, two):
	print "\tTesting registry"
	test_registry(one, two)
	print "\tTesting exclusive locks"
	test_exclusive(one, two)
	print "\tTesting shared locks"
	test_shared(one, two)
	print "\tTesting introspection"
	test_introspection(one, two)
	print "\tTesting orphaning of locks"
	test_orphan(one, two)

def test_unix():
	print "Testing UNIX Sockets (%s)" % options.unix
	one = gunix( options.unix )
	two = gunix( options.unix )
	test( one, two )

def test_tcp():
	print "Testing TCP Sockets (%s)" % options.tcp
	one = gtcp( options.tcp )
	two = gtcp( options.tcp )
	test( one, two )

def test_ws():
	print "Testing WebSockets (%s)" % options.ws
	one = gws( options.ws )
	two = gws( options.ws )
	test( one, two )

def test_unix_tcp():
	print "Testing UNIX Sockets (%s) and TCP Sockets (%s) mixed" % ( options.unix, options.tcp )
	one = gunix( options.unix )
	two = gtcp( options.tcp )
	test( one, two )

def test_unix_ws():
	print "Testing UNIX Sockets (%s) and WebSockets (%s) mixed" % ( options.unix, options.ws )
	one = gunix( options.unix )
	two = gws( options.ws )
	test( one, two )

def test_tcp_ws():
	print "Testing TCP Sockets (%s) and WebSockets (%s) mixed" % ( options.tcp, options.ws )
	one = gtcp( options.tcp )
	two = gws( options.ws )
	test( one, two )

def test_all():
	test_unix()
	test_tcp()
	test_ws()
	test_unix_tcp()
	test_unix_ws()
	test_tcp_ws()

ok = u"\t\t\033[92m\u2713\033[0m\t"
no = u"\t\t\033[91m\u2717\033[0m\t"
random_lock_string = ''.join(random.choice(string.ascii_uppercase + string.digits) for x in range(40))

test_all()
