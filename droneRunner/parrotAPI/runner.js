'use strict';

var RollingSpider = require('rolling-spider');
var temporal = require('temporal');
var rollingSpider = new RollingSpider();

rollingSpider.connect(function () {
  rollingSpider.setup(function () {
    rollingSpider.flatTrim();
    rollingSpider.startPing();
    rollingSpider.flatTrim();
    console.log("Prep for take off")
    temporal.queue([
      {
        delay: 5000,
        task: function () {
          rollingSpider.takeOff();
          rollingSpider.flatTrim();
        }
      },
      {
        delay: 5000,
        task: function () {
          console.log("Landing commenced")
          rollingSpider.land();
        }
      },
      {
        delay: 5000,
        task: function () {
          temporal.clear();
          console.log("Exiting process")
          process.exit(0);
        }
      }
    ]);
  });
});
