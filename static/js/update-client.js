/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

$(document).ready(function () {
  function clearClientSession() {
    // clear client session values
    $.ajax({
      url: "/api/v1/clear-update-client-session",
      type: "DELETE",
      contentType: "application/json",
    });
  }
  $("#fullTunnel").on("change", function () {
    serverCidr = $("#serverSupernet").attr("data-supernet");
    updateAllowedIPs();
  });

  $("#cancel").on("click", function (event) {
    event.preventDefault();
    clearClientSession();
    location.replace("/client");
  });

  var serverAddress = $("#server").attr("data-server-address") || "";
  var serverDns = serverAddress.split("/")[0];

  // pre-check "Use Server DNS" if current DNS matches the server IP
  if (serverDns && $("#dnsServers").val() === serverDns) {
    $("#useServerDns").prop("checked", true);
  }

  $("#useServerDns").change(function () {
    if ($(this).is(":checked")) {
      $("#dnsServers").val(serverDns);
    } else {
      $("#dnsServers").val("");
    }
  });

  const initName = $("#name").val();
  const initAddr = $("#address").val();
  const initIps = $("#allowedIps").val();
  const initDns = $("#dnsServers").val();
  const initFt = $("#fullTunnel").val();
  const initKl = $("#keepalive").val();

  $("#update-client-form").on("submit", function (event) {
    event.preventDefault();

    // change validation here
    const formChanged =
      normalize(initName) !== normalize($("#name").val()) ||
      normalize(initAddr) !== normalize($("#address").val()) ||
      normalize(initIps) !== normalize($("#allowedIps").val()) ||
      normalize(initDns) !== normalize($("#dnsServers").val()) ||
      normalize(initFt) !== normalize($("#fullTunnel").val()) ||
      normalize(initKl) !== normalize($("#keepalive").val());

    if (!formChanged) {
      showModal("Update client", "No changes detected. Nothing to update", {
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

    $.each(formArray, function (_, field) {
      formData[field.name] = field.value;
    });

    // override fullTunnel handling to follow obj requirements
    formData["fullTunnel"] = $("#fullTunnel").is(":checked");
    formData["publicKey"] = $("#publicKey").val();
    formData["privateKey"] = $("#privateKey").val();
    formData["clientId"] = $("#clientId").attr("data-client-id");
    formData["serverId"] = $("#server").attr("data-server-id");
    formData["dnsServers"] = $("#dnsServers").val();

    $.ajax({
      url: "/api/v1/update-client",
      type: "PUT",
      contentType: "application/json",
      data: JSON.stringify(formData),
      success: function (response) {
        showModal("Update client", response.message, {
          showCloseButton: false,
          showActionButton: true,
          status: response.status,
          onAction: function () {
            hideModal();
            location.replace("/client");
          },
        });
      },
      error: function (xhr, status, error) {
        defaultModalError("Update client", xhr, status);
      },
    });
  });
});
