/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

var serverCidr = ""; // predefine for later reference

function updateAllowedIPs() {
  var allowedIpsInput = $("#allowedIps");
  var currentValues = allowedIpsInput
    .val()
    .split(",")
    .map(function (item) {
      return item.trim();
    })
    .filter(Boolean); // split by commas trim whitespace and remove empty entries

  var serverCidrArray = serverCidr.split(",").map(function (item) {
    return item.trim();
  });

  if ($("#fullTunnel").is(":checked")) {
    // add each serverCidr to currentValues if not already present
    serverCidrArray.forEach(function (cidr) {
      if (!currentValues.includes(cidr)) {
        currentValues.push(cidr);
      }
    });
  } else {
    // remove all occurrences of each serverCidr from currentValues
    currentValues = currentValues.filter(function (item) {
      return !serverCidrArray.includes(item);
    });
  }

  // join values back into a string and set it as the new value
  allowedIpsInput.val(currentValues.join(", "));
}
