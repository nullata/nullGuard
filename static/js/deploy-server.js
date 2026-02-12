/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

$(document).ready(function () {
  function serverAction($btn, url, title) {
    var originalHtml = $btn.html();
    $btn.prop("disabled", true).html(
      '<i class="fas fa-spinner fa-spin me-2"></i>' + title.replace("&", "&amp;") + "..."
    );

    var formData = {};
    formData["serverId"] = $("#serverSelect").val();
    formData["interfaceName"] = $("#interfaceName").val();

    $.ajax({
      url: url,
      type: "POST",
      contentType: "application/json",
      data: JSON.stringify(formData),
      success: function (response) {
        $btn.prop("disabled", false).html(originalHtml);
        defaultModalSuccess(title, response);
      },
      error: function (xhr, status, error) {
        $btn.prop("disabled", false).html(originalHtml);
        defaultModalError(title, xhr, status);
      },
    });
  }

  $("#start").on("click", function (event) {
    event.preventDefault();
    serverAction($(this), "/api/v1/deploy-server", "Deploy & Start");
  });

  $("#stop").on("click", function (event) {
    event.preventDefault();
    serverAction($(this), "/api/v1/stop-server", "Stop Server");
  });

  $("#restart").on("click", function (event) {
    event.preventDefault();
    serverAction($(this), "/api/v1/restart-server", "Restarting");
  });
});
