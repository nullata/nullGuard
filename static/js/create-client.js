/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

$(document).ready(function () {
  var serverDns = "";

  function loadClientDefaultData(serverId) {
    if (serverId) {
      $.ajax({
        url: "/api/v1/build-client",
        method: "POST",
        contentType: "application/json",
        data: JSON.stringify({ serverId: serverId }),
        success: function (response) {
          if (!response.data) {
            // error here
            return;
          }

          var clientData = response.data.clientData;
          serverDns = response.data.serverDns || "";

          $("#address").val(clientData.address);
          $("#allowedIps").val(clientData.allowedIps);
          $("#keepalive").val(clientData.keepalive);
          serverCidr = clientData.serverSupernet; // adds value to initial definition in update-ips

          // reset DNS fields on server change
          $("#dnsServers").val("");
          $("#useServerDns").prop("checked", false);

          updateAllowedIPs();
        },
        error: function (xhr, status, error) {
          defaultModalError("Load server", xhr, status);
        },
      });
    }
  }

  $("#create-client-form").on("submit", function (event) {
    event.preventDefault();

    var formArray = $(this).serializeArray();
    var formData = {};

    // array to a JSON
    $.each(formArray, function (_, field) {
      formData[field.name] = field.value;
    });

    // override fullTunnel handling to follow obj requirements
    formData["fullTunnel"] = $("#fullTunnel").is(":checked");
    // add disabled fields
    formData["publicKey"] = $("#publicKey").val();
    formData["privateKey"] = $("#privateKey").val();
    // grab the server id
    formData["serverId"] = $("#serverSelect").val();
    // add dns servers
    formData["dnsServers"] = $("#dnsServers").val();

    $.ajax({
      url: "/api/v1/create-client",
      type: "POST",
      contentType: "application/json",
      data: JSON.stringify(formData),
      success: function (response) {
        showModal("Create client", response.message, {
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
        defaultModalError("Create client", xhr, status);
      },
    });
  });

  $("#fullTunnel").change(function () {
    updateAllowedIPs();
  });

  $("#useServerDns").change(function () {
    if ($(this).is(":checked")) {
      $("#dnsServers").val(serverDns);
    } else {
      $("#dnsServers").val("");
    }
  });

  $("#cancel").on("click", function (event) {
    event.preventDefault();
    location.replace("/client");
  });

  $("#serverSelect").change(function () {
    var serverId = $(this).val();
    loadClientDefaultData(serverId); // fetch data when server is selected
  });

  var initialServerId = $("#serverSelect").val(); // currenttly selected server id
  if (initialServerId) {
    loadClientDefaultData(initialServerId); // fetch data on initial page load
  }
});
