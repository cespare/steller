# Steller

Steller is an HTTP benchmarking tool.

Status: Completely WIP. Pre-alpha. Not safe for consumption.

## TODO

* Randomly choose when to generate requests (Poisson process?) (Instead of having the user choose # of threads
  and concurrency level, like ab or wrk).
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
