/*
Package tablecloth enables creating HTTP servers that support zero-downtime restarts. It
wraps functions from net/http to listen for a restart signal, and then gracefully restart the
application without dropping any requests.

When the application process recieves a SIGHUP, it will start a temporary child process to continue
serving requests while the original process re-exec's itself.  Once this has happened, the temporary
child is stopped.  This means that after restart, the process retains the same process id, and
therefore plays nicely with process supervisors like upstart.

This implementation is based on an approach described here:
http://blog.nella.org/zero-downtime-upgrades-of-tcp-servers-in-go/
*/
package tablecloth
