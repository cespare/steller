# Steller

Steller is an HTTP benchmarking tool.

Status: Completely WIP. Pre-alpha. Not safe for consumption.

## Notes about the configuration format

* It's JSON -- see conf.json for an example
* Requests are chosen randomly from the list. Use `"weight"` in each URL to weight the probabilities.
* Requests can be in the configuration and/or in a separate file. Use `"requests_file"` to specify the
  filename.
* The body of the request can be given with `"body"`, or in a separate file using `"body_file"`.
* `"target_qps"` is an integer, or else `"unlimited"` to have each worker make requests as quickly as
  possible.
* Use `"reporting_stats"` to specify quantile divisions and latency bucket breakdowns.

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
* Warmup before test starts
* Handle passing multiple different hosts? (This should probably not work.)
* Write comparisons to ab, wrk, siege, jmeter, tsung
