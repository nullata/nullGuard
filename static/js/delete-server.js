/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

$(document).ready(function () {
  $("#delete").on("click", function (event) {
    event.preventDefault();

    var formData = {};
    formData["serverId"] = $("#serverSelect").val();
    formData["interfaceName"] = $("#interfaceName").val();

    showModal(
      "Delete Server",
      "Are you sure you want to delete server " +
        $("#serverSelect option:selected").text() +
        " and all of its configured clients?",
      {
        showCloseButton: true,
        showActionButton: true,
        actionName: "Delete",
        actionStyle: "btn btn-danger",
        status: "error",
        onAction: function () {
          $.ajax({
            url: "/api/v1/delete-server",
            type: "DELETE",
            contentType: "application/json",
            data: JSON.stringify(formData),
            success: function (response) {
              hideModal();
              location.reload();
            },
            error: function (xhr, status, error) {
              defaultModalError("Delete server", xhr, status);
            },
          });
        },
      },
    );
  });
});
