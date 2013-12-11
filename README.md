# Steller

Steller is an HTTP benchmarking tool.

Status: Completely WIP. Pre-alpha. Not safe for consumption.

## TODO

* Allow the user to ramp up the request rate. For instance, maybe specify start rate, end rate, and number of
  increases.
* Show stats over time with plotinum.
* Distribution
  - Take a list of servers (must be same os/arch, must have passwordless SSH)
  - SFTP own binary onto machines and coordinate using net/rpc(?)
  - Slaves send back all the results and they're all tallied on the master
* Construct a simple test at command-line directly, a la ab/wrk.
* Give expected responses?
* User-configurable timeout
* Remove all allocations from the critical (timed) path. Call `runtime.GC()` before timing starts.
* Warm-up before test starts
* Handle passing multiple different hosts? (This should probably not work.)
* Write comparisons to ab, wrk, siege, jmeter, tsung

## Measurements

* qps over time
  - broken down by success vs. failure
  - successful requests broken down by status code
