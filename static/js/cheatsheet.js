/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

// delegated click handler for any clickable info-tooltip on any page.
// fetches the cheatsheet body from /help/<key> and drops it into the
// shared #modal. backdrop:true and keyboard:true override showModal's
// default static-backdrop behavior so docs feel dismissable, not
// confirm-blocking.
$(document).on("click", ".info-tooltip-clickable", function (event) {
  // stop the click from bubbling to whatever the tooltip is decorating
  // (e.g. a label that would steal focus to its input).
  event.preventDefault();
  event.stopPropagation();

  var key = $(this).data("cheatsheet");
  var title = $(this).data("cheatsheet-title") || "Cheatsheet";

  $.get("/help/" + encodeURIComponent(key))
    .done(function (html) {
      showModal(title, html, {
        showCloseButton: true,
        backdrop: true,
        keyboard: true,
        modalClass: "modal-cheatsheet",
      });
    })
    .fail(function () {
      showModal(title, "Cheatsheet not found.", {
        showCloseButton: true,
        status: "error",
        backdrop: true,
        keyboard: true,
        modalClass: "modal-cheatsheet",
      });
    });
});
