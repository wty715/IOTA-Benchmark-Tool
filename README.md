# IOTA-Benchmark-Tool
Including four parts:
1. IOTA Full Node Implementation: IRI-1.5.5.jar and its configuration
2. A script for automaticlly sending transactions
3. A script for sending double-spending transactions
4. A monitor to do the real-time monitoring

For auto-wallet (double-spending), run it with following steps:
1. cd auto-wallet (double-spending)
2. npm install
3. ./start.sh

For monitor, run it with following steps:
1. cd monitor
2. go build
3. ./start.sh