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
* Request URL options
  * URL at command-line
  * File with a list of URLs?
  * syntax for specifying POST bodies, headers, etc
  * Expected responses?
* User-configurable timeout
* Cancel outstanding requests?
* Remove all allocations from the critical (timed) path.
* Warm-up before test starts
