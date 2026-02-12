/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

$(document).ready(function () {
  $("#create-server-form").on("submit", function (event) {
    event.preventDefault();

    var formArray = $(this).serializeArray();
    var formData = {};

    // array to a JSON
    $.each(formArray, function (_, field) {
      formData[field.name] = field.value;
    });

    // add disabled fields
    formData["publicKey"] = $("#publicKey").val();
    formData["privateKey"] = $("#privateKey").val();

    $.ajax({
      url: "/api/v1/create-server",
      type: "POST",
      contentType: "application/json",
      data: JSON.stringify(formData),
      success: function (response) {
        showModal("Create server", response.message, {
          showCloseButton: false,
          showActionButton: true,
          status: response.status,
          onAction: function () {
            hideModal();
            location.replace("/server");
          },
        });
      },
      error: function (xhr, status, error) {
        defaultModalError("Create server", xhr, status);
      },
    });
  });

  $("#cancel").on("click", function (event) {
    event.preventDefault();
    location.replace("/server");
  });
});
