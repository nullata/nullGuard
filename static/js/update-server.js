/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

$(document).ready(function () {
  $("#update-server-form").on("submit", function (event) {
    event.preventDefault();

    const formChanged =
      normalize($("#interfaceName").val()) !==
        normalize(initialData.interfaceName) ||
      normalize($("#comment").val()) !== normalize(initialData.comment) ||
      normalize($("#address").val()) !== normalize(initialData.address) ||
      normalize($("#port").val()) !== normalize(initialData.port) ||
      normalize($("#postUp").val()) !== normalize(initialData.postUp) ||
      normalize($("#postDown").val()) !== normalize(initialData.postDown) ||
      normalize($("#wanAddress").val()) !== normalize(initialData.wanAddress) ||
      normalize($("#supernetCidr").val()) !==
        normalize(initialData.supernetCidr) ||
      normalize($("#defaultKeepAlive").val()) !==
        normalize(initialData.defaultKeepAlive);

    if (!formChanged) {
      showModal("Update server", "No changes detected. Nothing to update", {
        showCloseButton: true,
        status: "error",
        onClose: function () {
          hideModal();
        },
      });
      return;
    }

    var formArray = $(this).serializeArray();
    var formData = {};

    // array to json
    $.each(formArray, function (_, field) {
      formData[field.name] = field.value;
    });

    // manually add disabled fields
    formData["serverId"] = $("#serverSelect").val();
    formData["publicKey"] = $("#publicKey").val();
    formData["privateKey"] = $("#privateKey").val();

    $.ajax({
      url: "/api/v1/update-server",
      type: "PUT",
      contentType: "application/json",
      data: JSON.stringify(formData),
      success: function (response) {
        defaultModalSuccess("Update server", response);
      },
      error: function (xhr, status, error) {
        defaultModalError("Update server", xhr, status);
      },
    });
  });
});
