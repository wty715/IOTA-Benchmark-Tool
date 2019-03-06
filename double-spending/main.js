const child_process = require('child_process')

const config = {
  'address'   : require('./config.json').address,
  'address2'  : require('./config.json').address2
}

var p = child_process.fork('double.js', [config.address], {})
p.on('exit', code => {
  var timeStamp = new Date().toISOString()
  console.log('[' + timeStamp + ']: ' + '[Normal Transfer] finished. Code: ' + code)
})
p.on('message', message => {
  var timeStamp = new Date().toISOString()
  if (message.type == 'error') {
    console.log('[' + timeStamp + ']: ' + '[Normal Transfer] error! Reason: ' + message.msg)
  }
  else if (message.type == 'log') {
    console.log('[' + timeStamp + ']: ' + '[Normal Transfer] sent: ' + message.msg)
  }
})

var p2 = child_process.fork('double.js', [config.address2], {})
p2.on('exit', code => {
  var timeStamp = new Date().toISOString()
  console.log('[' + timeStamp + ']: ' + '[Double-Spending] finished. Code: ' + code)
})
p2.on('message', message => {
  var timeStamp = new Date().toISOString()
  if (message.type == 'error') {
    console.log('[' + timeStamp + ']: ' + '[Double-Spending] error! Reason: ' + message.msg)
  }
  else if (message.type == 'log') {
    console.log('[' + timeStamp + ']: ' + '[Double-Spending] sent: ' + message.msg)
  }
})
