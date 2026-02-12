/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

$(document).ready(function () {
  $("#login-form").submit(function (e) {
    e.preventDefault();

    const formData = {
      username: $("#username").val().trim(),
      password: $("#password").val(),
    };

    $.ajax({
      url: "/api/v1/login",
      type: "POST",
      contentType: "application/json",
      data: JSON.stringify(formData),
      success: function (response) {
        window.location.href = "/";
      },
      error: function (xhr) {
        const response = xhr.responseJSON;
        const message =
          response && response.message ? response.message : "Login failed";
        showModal("Login Failed", message, {
          showCloseButton: true,
          status: "error",
        });
      },
    });
  });

  $("#show-hint-link").click(function (e) {
    e.preventDefault();

    const username = $("#username").val().trim();
    if (!username) {
      showModal("Username Required", "Please enter your username first", {
        showCloseButton: true,
        status: "error",
      });
      return;
    }

    $.ajax({
      url: "/api/v1/get-password-hint?username=" + encodeURIComponent(username),
      type: "GET",
      success: function (response) {
        if (response.data && response.data.hint) {
          $("#hint-text").text(response.data.hint);
          $("#hint-display").show();
        } else {
          $("#hint-text").text("No password hint available");
          $("#hint-display").show();
        }
      },
      error: function (xhr) {
        const response = xhr.responseJSON;
        const message =
          response && response.message
            ? response.message
            : "Failed to retrieve password hint";
        showModal("Error", message, {
          showCloseButton: true,
          status: "error",
        });
      },
    });
  });
});
