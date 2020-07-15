# IOTA-Benchmark-Tool

A benchmark tool for IOTA blockchain. The automatic transaction initiator is implemented with multi-threading NodeJS, utilizing APIs provided by IOTA foundation. Each initiator will automatically send transactions with pre-configured sender’s seed and receiver’s address in ***config.json***.

The monitor is implemented with Golang for high performance.

Utilizing these tools, you can exhibit some unique characteristics of IOTA from several aspects such as performance, security, etc.

---

Including four parts:
+ IOTA Full Node Implementation: IRI-1.5.5.jar and its configuration
+ A script for automaticlly sending transactions
+ A script for sending double-spending transactions
+ A monitor to do the real-time monitoring

For auto-wallet (double-spending), run it with following steps:
+ $cd auto-wallet (double-spending)
+ $npm install
+ $./start.sh

For monitor, run it with following steps:
+ $cd monitor
+ $go build
+ $./start.sh

---

Hints: The visualization of private IOTA network can be found at https://www.youtube.com/channel/UCj0muwSgKr2bn2BybLkw3kQ?view_as=subscriber
