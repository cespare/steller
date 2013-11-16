# Steller

Steller is an HTTP benchmarking tool.

Status: Completely WIP. Pre-alpha. Not safe for consumption.

## TODO

* Allow the user to ramp up the request rate. For instance, maybe specify start rate, end rate, and number of
  increases.
* Stats over time with plotinum.
  * qps
  * success and failure qps
  * Mean (and other quantiles) response times
  * Counts of response status codes
* Distribution
  * Take a list of servers (must be same os/arch, must have passwordless SSH)
  * SFTP own binary onto machines and coordinate using net/rpc(?)
  * Slaves send back all the results and they're all tallied on the master
* Construct a simple test at command-line directly, a la ab/wrk.
* Provide a url file instead of a list in the config
  * Can list the common fields at the top, differing ones in the list
* Specify request headers
* Give expected responses?
* User-configurable timeout
* Cancel outstanding requests?
* Remove all allocations from the critical (timed) path. Call `runtime.GC()` before timing starts.
* Warm-up before test starts
* Handle passing multiple different hosts? (This should probably not work.)
* Think about how to set `GOMAXPROCS`.
* Write comparisons to ab, wrk, siege, jmeter, tsung
