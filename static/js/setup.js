/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

$(document).ready(function () {
  $("#setup-form").submit(function (e) {
    e.preventDefault();

    const formData = {
      username: $("#username").val().trim(),
      password: $("#password").val(),
      password_confirm: $("#password_confirm").val(),
      password_hint: $("#password_hint").val().trim() || null,
    };

    $.ajax({
      url: "/api/v1/setup",
      type: "POST",
      contentType: "application/json",
      data: JSON.stringify(formData),
      success: function (response) {
        const message =
          response.message || "Admin account created successfully";
        showModal("Success", message, {
          showActionButton: true,
          status: "success",
          onAction: function () {
            window.location.href = "/login";
          },
        });
      },
      error: function (xhr) {
        const response = xhr.responseJSON;
        const message =
          response && response.message
            ? response.message
            : "Failed to create admin account";
        showModal("Error", message, {
          showCloseButton: true,
          status: "error",
        });
      },
    });
  });
});
