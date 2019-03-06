const config = {
  'host'                : require('./config.json').host,
  'port'                : require('./config.json').port,
  'seed'                : require('./config.json').seed,
  'amount'              : require('./config.json').amount,
  'language'            : require('./config.json').language,
  'tag'                 : require('./config.json').tag,
  'depth'               : require('./config.json').depth,
  'minWeightMagnitude'  : require('./config.json').minWeightMagnitude
}
const IOTA = require('iota.lib.js')
const iota = new IOTA({
  "host": config.host,
  "port": config.port
})

/* check node infomation
iota.api.getNodeInfo((error, nodeInfo) => {
  if (error) {
    console.log(error)
    return
  }
  else {
    console.log(nodeInfo)
    return
  }
})
*/

//console.log("Prepare transfer: " + process.argv[2] + " -> " + config.amount)
process.send({'type':'log','msg':"Prepare transfer: " + process.argv[2] + " -> " + config.amount})

getUnspentInputs(config.seed, 0, config.amount, function(error, inputs) {
  if (error && error.message !== 'Not enough balance') {
    //console.log(error)
    process.send({'type':'error','msg':error.toString()})
    //return
    process.exit()
  } else if (inputs.allBalance < config.amount) { // Not enough balance
    //console.log('Not enough balance!')
    process.send({'type':'error','msg':'Not enough balance!'})
    //return
    process.exit()
  } else if (inputs.totalBalance < config.amount) { // Some used addresses still have balance
    //console.log('Some used addresses still have balance!')
    //console.log('Key Reuse Error!')
    process.send({'type':'error','msg':'Some used addresses still have balance!'})
    //return
    process.exit()
  }

  //console.log('Unspent input txs got.')
  process.send({'type':'log','msg':'Unspent input txs got.'})

  var transfers = [{"address": process.argv[2], "value": config.amount, "message": "", "tag": config.tag}]
  var outputsToCheck = transfers.map(transfer => iota.utils.noChecksum(transfer.address)) // delete the last 9 bytes checksum
  var exptectedOutputsLength = outputsToCheck.length
  filterSpentAddresses(outputsToCheck).then(filtered => { // check if the receiving address was used or not
    if (filtered.length !== exptectedOutputsLength) { // the receiving address was used before
      //console.log('The receiving address was used before!')
      //console.log('Key Reuse Error!')
      process.send({'type':'error','msg':'The receiving address was used before!'})
      //return
      process.exit()
    }

    //console.log('Output address checked.')
    process.send({'type':'log','msg':'Output address checked.'})

    iota.api.prepareTransfers(config.seed, transfers, {"inputs": inputs.inputs}, function(error, trytes) {
      if (error) {
        //console.log(error)
        process.send({'type':'error','msg':error.toString()})
        //return
        process.exit()
      }
      var sentToInputs = false
      trytes.forEach(transactionTrytes => {
        var tx = iota.utils.transactionObject(transactionTrytes) // transfer to normal format
        if (inputs.inputs.findIndex(input => tx.value > 0 && input.address === tx.address) !== -1) {
          sentToInputs = true // for each transaction in bundle, check if it is both receiver and senter
        }
      })
      if (sentToInputs) {
        //console.log('The tranfer destination and source were conflicting!')
        process.send({'type':'error','msg':'The tranfer destination and source were conflicting!'})
        //return
        process.exit()
      }

      //console.log('The whole tranfer checked.')
      process.send({'type':'log','msg':'The whole tranfer checked.'})

      iota.api.getTransactionsToApprove(config.depth, {}, function(error, toApprove) {
        if (error) {
          console.log(error)
          return
        }

        //console.log('Two tips selected.')
        process.send({'type':'log','msg':'Two tips selected.'})

        // attach to tangle - do pow
        iota.api.attachToTangle(toApprove.trunkTransaction, toApprove.branchTransaction, config.minWeightMagnitude, trytes, function(error, attached) {
          if (error) {
            //console.log(error)
            process.send({'type':'error','msg':error.toString()})
            //return
            process.exit()
          }

          //console.log('Attached to tangle - POW done.')
          process.send({'type':'log','msg':'Attached to tangle - POW done.'})

          // Broadcast and store tx
          iota.api.storeAndBroadcast(attached, function(error, success) {
            if (error) {
              //console.log(error)
              process.send({'type':'error','msg':error.toString()})
              //return
              process.exit()
            }
            var finalTxs = []
            attached.forEach(function(trytes) {
                finalTxs.push(iota.utils.transactionObject(trytes))
            })
            //console.log(finalTxs)
            //console.log("Transfer normally: Success")
            process.send({'type':'log','msg':'Transfer normally: Success'})
            //return
            process.exit()
          })
        })
      })
    })
  }).catch(error => {
    //console.log(error)
    process.send({'type':'error','msg':error.toString()})
    //return
    process.exit()
  })
})

function filterSpentAddresses (addresses) {
  return new Promise((resolve, reject) => {
    iota.api.wereAddressesSpentFrom( // only return unused addresses
      iota.valid.isArrayOfHashes(addresses) ? addresses : addresses.map(address => address.address),
      (err, wereSpent) => err ? reject(err) : resolve(addresses.filter((address, i) => !wereSpent[i]))
    )
  })
}

function getUnspentInputs (seed, start, threshold, inputs, callback) {
  if (arguments.length === 4) {
    callback = arguments[3]
    inputs = {inputs: [], totalBalance: 0, allBalance: 0} // totalBalance: with all unused addresses
  }

  //console.log('Preparing to get all input txs......')
  process.send({'type':'log','msg':'Preparing to get all input txs......'})

  iota.api.getInputs(seed, {start: start, threshold: threshold}, (err, res) => { // all addresses with balance
    if (err) {
      callback(err, inputs)
      return
    }

    //console.log('All input txs got.')
    process.send({'type':'log','msg':'All input txs got.'})
    
    inputs.allBalance += res.inputs.reduce((sum, input) => sum + input.balance, 0) // calculate the sum of balance in all addresses
    filterSpentAddresses(res.inputs).then(filtered => {
      var collected = filtered.reduce((sum, input) => sum + input.balance, 0) // the sum of balance in all unused addresses
      var diff = threshold - collected
      if (diff > 0) { // still need other inputs
        var ordered = res.inputs.sort((a, b) => a.keyIndex - b.keyIndex).reverse()
        var end = ordered[0].keyIndex
        getUnspentInputs(seed, end + 1, diff, {inputs: inputs.inputs.concat(filtered), totalBalance: inputs.totalBalance + collected, allBalance: inputs.allBalance}, callback)
      } else {
        callback(null, {inputs: inputs.inputs.concat(filtered), totalBalance: inputs.totalBalance + collected, allBalance: inputs.allBalance})
      }
    }).catch(err => callback(err, inputs))
  })
}
