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

// add or remove a single CIDR from the AllowedIps field. used by any cidr-toggle-checkbox
function toggleAllowedIp(cidr, checked) {
  var allowedIpsInput = $("#allowedIps");
  var currentValues = allowedIpsInput
    .val()
    .split(",")
    .map(function (item) {
      return item.trim();
    })
    .filter(Boolean);

  cidr = (cidr || "").trim();
  if (!cidr) return;

  if (checked) {
    if (!currentValues.includes(cidr)) {
      currentValues.push(cidr);
    }
  } else {
    currentValues = currentValues.filter(function (item) {
      return item !== cidr;
    });
  }

  allowedIpsInput.val(currentValues.join(", "));
}

// render an unchecked checkbox per CIDR into $container, returning whether anything was rendered.
// idPrefix scopes element ids so multiple groups on the same page don't collide.
function renderCidrCheckboxes($container, cidrs, idPrefix) {
  $container.empty();
  if (!cidrs || cidrs.length === 0) {
    return false;
  }
  cidrs.forEach(function (cidr, idx) {
    var id = idPrefix + "-" + idx;
    var $wrap = $("<div>").addClass("form-check");
    var $cb = $("<input>")
      .attr({ type: "checkbox", id: id, "data-cidr": cidr })
      .addClass("form-check-input cidr-toggle-checkbox");
    var $lbl = $("<label>")
      .attr("for", id)
      .addClass("form-check-label")
      .text(cidr);
    $wrap.append($cb).append($lbl);
    $container.append($wrap);
  });
  return true;
}

// single delegated handler for every cidr-toggle-checkbox on the page
$(document).on("change", ".cidr-toggle-checkbox", function () {
  toggleAllowedIp($(this).data("cidr"), $(this).is(":checked"));
});
