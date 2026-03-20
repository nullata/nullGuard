/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

$(document).ready(function () {
  function adjustTextareaHeight(textarea) {
    $(textarea).css("height", "auto"); // reset height
    $(textarea).css("height", textarea.scrollHeight + "px"); // set height based on scrollHeight
  }

  function updateServerStatus(isActive) {
    if (isActive) {
      $("#serverStatusText").text("Active");
      $("#serverStatusDot").removeClass("inactive").addClass("active");
      $("#start").addClass("d-none");
      $("#stop").removeClass("d-none");
      $("#restart").removeClass("d-none");
    } else {
      $("#serverStatusText").text("Inactive");
      $("#serverStatusDot").removeClass("active").addClass("inactive");
      $("#stop").addClass("d-none");
      $("#restart").addClass("d-none");
      $("#start").removeClass("d-none");
    }
  }

  function loadServerData(serverId) {
    if (serverId) {
      $.ajax({
        url: "/api/v1/fetch-server",
        method: "POST",
        contentType: "application/json",
        data: JSON.stringify({ serverId: serverId }),
        success: function (response) {
          $("#interfaceName").val(response.data.InterfaceName);
          $("#comment").val(response.data.Comment);
          $("#address").val(response.data.Address);
          $("#port").val(response.data.Port);
          $("#publicKey").val(response.data.PublicKey);
          $("#privateKey").val(response.data.PrivateKey);
          $("#postUp").val(response.data.PostUp);
          $("#postDown").val(response.data.PostDown);
          $("#wanAddress").val(response.data.WANAddress);
          $("#supernetCidr").val(response.data.SupernetCidr);
          $("#defaultKeepAlive").val(response.data.DefaultKeepalive);

          initialData = {
            interfaceName: response.data.InterfaceName,
            comment: response.data.Comment,
            address: response.data.Address,
            port: response.data.Port,
            publicKey: response.data.PublicKey,
            privateKey: response.data.PrivateKey,
            postUp: response.data.PostUp,
            postDown: response.data.PostDown,
            wanAddress: response.data.WANAddress,
            supernetCidr: response.data.SupernetCidr,
            defaultKeepAlive: response.data.DefaultKeepalive,
          };

          // set auto-restart toggle state
          $("#autoRestart").prop("checked", response.data.AutoRestart);

          // adjust height on page load for all matching textareas
          adjustTextareaHeight($("#postUp")[0]);
          adjustTextareaHeight($("#postDown")[0]);

          // update server status and buttons
          updateServerStatus(response.data.IsActive);
        },
        error: function (xhr, status, error) {
          defaultModalError("Load server", xhr, status);
        },
      });
    }
  }

  $("#serverSelect").change(function () {
    var serverId = $(this).val();
    loadServerData(serverId); // fetch data when server is selected
  });

  function refreshServerStatus() {
    var serverId = $("#serverSelect").val();
    if (!serverId) return;
    $.ajax({
      url: "/api/v1/fetch-server",
      method: "POST",
      contentType: "application/json",
      data: JSON.stringify({ serverId: serverId }),
      success: function (response) {
        updateServerStatus(response.data.IsActive);
      },
    });
  }

  var initialServerId = $("#serverSelect").val(); // currenttly selected server id
  if (initialServerId) {
    loadServerData(initialServerId); // fetch data on initial page load
  }

  // toggle auto-restart setting
  $("#autoRestart").change(function () {
    var serverId = $("#serverSelect").val();
    if (!serverId) return;
    $.ajax({
      url: "/api/v1/toggle-auto-restart",
      method: "POST",
      contentType: "application/json",
      data: JSON.stringify({
        serverId: serverId,
        autoRestart: $(this).is(":checked"),
      }),
      error: function (xhr, status) {
        defaultModalError("Toggle auto-restart", xhr, status);
      },
    });
  });

  setInterval(refreshServerStatus, 30000); // refresh server status every 30 seconds

  // rotate chevron when collapse is toggled
  var configCollapse = document.getElementById("serverConfigCollapse");
  if (configCollapse) {
    configCollapse.addEventListener("show.bs.collapse", function () {
      document.getElementById("serverConfigChevron").style.transform =
        "rotate(0deg)";
    });
    configCollapse.addEventListener("hide.bs.collapse", function () {
      document.getElementById("serverConfigChevron").style.transform =
        "rotate(-180deg)";
    });
  }
});
