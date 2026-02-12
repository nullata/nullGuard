/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

$(document).ready(function () {
  $("#logout-link").click(function (e) {
    e.preventDefault();
    $.ajax({
      url: "/api/v1/logout",
      type: "POST",
      success: function () {
        window.location.href = "/login";
      },
      error: function () {
        showModal("Logout Failed", "Failed to logout. Please try again.", {
          showCloseButton: true,
          status: "error",
        });
      },
    });
  });

  // auto-logout when the session expires
  var maxAgeMeta = document.querySelector('meta[name="session-max-age"]');
  if (maxAgeMeta) {
    var maxAge = parseInt(maxAgeMeta.getAttribute("content"), 10);
    if (maxAge > 0) {
      setTimeout(function () {
        window.location.href = "/login";
      }, maxAge * 1000);
    }
  }
});
