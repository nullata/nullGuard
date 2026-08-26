/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

function formatBytes(bytes) {
  if (bytes === 0) return "0 B";
  var units = ["B", "KiB", "MiB", "GiB", "TiB"];
  var i = Math.floor(Math.log(bytes) / Math.log(1024));
  if (i >= units.length) i = units.length - 1;
  var value = bytes / Math.pow(1024, i);
  return (i === 0 ? value : value.toFixed(2)) + " " + units[i];
}

function ipv4ToInt(addr) {
  var ip = String(addr || "").split("/")[0];
  var parts = ip.split(".");
  if (parts.length !== 4) return Number.MAX_SAFE_INTEGER;
  return parts.reduce(function (acc, p) {
    return acc * 256 + (parseInt(p, 10) || 0);
  }, 0);
}

function sortClientList() {
  var sortBy = $("#sortClient").val() || "address";
  var $list = $("#clientList");
  var items = $list.children("li").not(".no-items-mt").detach().toArray();
  items.sort(function (a, b) {
    var $a = $(a), $b = $(b);
    if (sortBy === "active") {
      var aActive = parseInt($a.attr("data-active"), 10) || 0;
      var bActive = parseInt($b.attr("data-active"), 10) || 0;
      if (aActive !== bActive) return bActive - aActive;
      var aHs = parseInt($a.attr("data-handshake"), 10) || 0;
      var bHs = parseInt($b.attr("data-handshake"), 10) || 0;
      return bHs - aHs;
    }
    if (sortBy === "traffic") {
      var aT = parseInt($a.attr("data-traffic"), 10) || 0;
      var bT = parseInt($b.attr("data-traffic"), 10) || 0;
      return bT - aT;
    }
    if (sortBy === "name") {
      return ($a.attr("data-name") || "").localeCompare($b.attr("data-name") || "");
    }
    return ipv4ToInt($a.attr("data-address")) - ipv4ToInt($b.attr("data-address"));
  });
  $list.append(items);
}

function formatHandshakeAge(lastHandshake) {
  if (!lastHandshake || lastHandshake <= 0) return "Last handshake: never";
  var seconds = Math.max(0, Math.floor(Date.now() / 1000 - lastHandshake));
  if (seconds < 60) return "Last handshake: " + seconds + "s ago";
  var minutes = Math.floor(seconds / 60);
  if (minutes < 60) return "Last handshake: " + minutes + "m ago";
  var hours = Math.floor(minutes / 60);
  if (hours < 24) return "Last handshake: " + hours + "h ago";
  var days = Math.floor(hours / 24);
  return "Last handshake: " + days + "d ago";
}

// show qr code in a modal
function showQRCodeModal(serverId, clientId, clientName) {
  const qrCodeUrl = `/api/v1/client/${serverId}/${clientId}/qrcode`;
  const modalContent = `
        <div class="text-center">
            <p>Scan this QR code with your WireGuard mobile app</p>
            <img src="${qrCodeUrl}" alt="QR Code" class="img-fluid" style="max-width: 300px;">
        </div>
    `;

  showModal(`QR Code - ${clientName}`, modalContent, {
    showCloseButton: true,
    showActionButton: false,
    status: "info",
  });
}

// download client configuration as zip
function downloadClientConfig(serverId, clientId) {
  const downloadUrl = `/api/v1/client/${serverId}/${clientId}/download`;
  window.location.href = downloadUrl;
}

$(document).ready(function () {
  // valid values for the sort dropdown, used to validate the stored value
  var CLIENT_SORT_VALUES = ["address", "name", "active", "traffic"];

  // restore the previously saved sort option (falls back to the default "address")
  function initClientSort() {
    var $select = $("#sortClient");
    if (!$select.length) return;
    var saved = localStorage.getItem("clientSort");
    if (saved && CLIENT_SORT_VALUES.indexOf(saved) !== -1) {
      $select.val(saved);
    }
  }
  initClientSort();

  function deleteClient(clientId, clientName, serverId) {
    var server = $("#serverSelect");
    if (server.val() === serverId) {
      showModal(
        "Delete Client",
        "Are you sure you want to delete client " +
          clientName +
          " for server " +
          $("#serverSelect option:selected").text() +
          "?",
        {
          showCloseButton: true,
          showActionButton: true,
          actionName: "Delete",
          actionStyle: "btn btn-danger",
          status: "error",
          onAction: function () {
            var data = {};
            data["clientId"] = clientId;
            data["serverId"] = serverId;
            $.ajax({
              url: "/api/v1/delete-client",
              type: "DELETE",
              contentType: "application/json",
              data: JSON.stringify(data),
              success: function (response) {
                hideModal();
                location.reload();
              },
              error: function (xhr, status, error) {
                defaultModalError("Delete client", xhr, status);
              },
            });
          },
        },
      );
    }
  }

  function buildDefaultListItem(message) {
    return $("<li>").text(message).addClass("text-center no-items-mt");
  }
  function loadServerClients(serverId, options = {}) {
    var data = {};
    data["serverId"] = serverId;
    $.ajax({
      url: "/api/v1/load-clients",
      type: "POST",
      contentType: "application/json",
      data: JSON.stringify(data),
      success: function (response) {
        $("#clientList").empty();
        if (response.data) {
          response.data.forEach((client) => {
            const $card = $("<li>").addClass("card mb-3");
            const $cardBody = $("<div>").addClass("card-body");
            // create a container for the title and buttons
            const $headerContainer = $("<div>").addClass(
              "d-flex justify-content-between align-items-center",
            );
            const $editIcon = $("<i>")
              .addClass("fas fa-pencil ms-2 edit-icon");
            const $statusDot = $("<span>")
              .addClass("status-dot inline " + (client.isConnected ? "active" : "inactive"))
              .attr("title", client.isConnected
                ? "Handshake within the last 3 minutes"
                : "No handshake within the last 3 minutes");
            const $cardTitle = $("<h5>")
              .addClass("card-title d-inline mb-0");
            const $cardTitleLink = $("<a>")
              .attr("href", "#")
              .addClass("client-edit-link")
              .attr("title", "Edit Client")
              .text(client.name)
              .append($editIcon)
              .on("click", function (event) {
                event.preventDefault();
                var editData = {};
                editData["clientId"] = client.clientId;
                editData["serverId"] = serverId;
                $.ajax({
                  url: "/api/v1/set-update-client-session",
                  type: "POST",
                  data: JSON.stringify(editData),
                  contentType: "application/json",
                  success: function () {
                    window.location.href = "/update-client";
                  },
                  error: function (xhr, status, error) {
                    defaultModalError("Edit client", xhr, status);
                  },
                });
              });
            $cardTitle.append($cardTitleLink);

            // button group
            const $buttonGroup = $("<div>").addClass("btn-group");

            // delete button
            const $deleteButton = $("<button>")
              .addClass("btn btn-client-action btn-client-delete btn-md")
              .attr("type", "button")
              .attr("title", "Delete Client")
              .append($("<i>").addClass("fas fa-trash"))
              .on("click", function (event) {
                event.preventDefault();
                deleteClient(client.clientId, client.name, serverId);
              });

            // qr code button
            const $qrButton = $("<button>")
              .addClass("btn btn-client-action btn-client-qr btn-md")
              .attr("type", "button")
              .attr("title", "Show QR Code")
              .append($("<i>").addClass("fas fa-qrcode"))
              .on("click", function (event) {
                event.preventDefault();
                showQRCodeModal(serverId, client.clientId, client.name);
              });

            // download button
            const $downloadButton = $("<button>")
              .addClass("btn btn-client-action btn-client-download btn-md")
              .attr("type", "button")
              .attr("title", "Download Config")
              .append($("<i>").addClass("fas fa-download"))
              .on("click", function (event) {
                event.preventDefault();
                downloadClientConfig(serverId, client.clientId);
              });

            // append buttons to button group
            $buttonGroup
              .append($qrButton)
              .append($downloadButton)
              .append($deleteButton);

            // wrap dot and title together so justify-content-between only splits title group vs buttons
            const $titleGroup = $("<div>").addClass("d-flex align-items-center")
              .append($statusDot)
              .append($cardTitle);

            // append title group and button group to header container
            $headerContainer.append($titleGroup).append($buttonGroup);

            // text content
            const $cardText = $("<p>").addClass("card-text");
            $cardText
              .append($("<strong>").text("Address CIDR: "))
              .append(document.createTextNode(client.address))
              .append("<br>");
            $cardText
              .append($("<strong>").text("Allowed IPs: "))
              .append(document.createTextNode(client.allowedIps))
              .append("<br>");
            $cardText
              .append($("<strong>").text("Keepalive: "))
              .append(document.createTextNode(client.keepalive))
              .append("<br>");

            // traffic stats
            const $trafficRow = $("<div>")
              .addClass("d-flex justify-content-end text-muted client-traffic")
              .append($("<span>").text(formatHandshakeAge(client.lastHandshake)))
              .append($("<span>").addClass("ms-3").append($("<i>").addClass("fas fa-arrow-up client-traffic-icon me-1")).append(formatBytes(client.transferRx)))
              .append($("<span>").addClass("ms-3").append($("<i>").addClass("fas fa-arrow-down client-traffic-icon me-1")).append(formatBytes(client.transferTx)));

            // append elements to their respective parents
            $cardBody.append($headerContainer).append($cardText).append($trafficRow);
            $card.append($cardBody);
            $card.attr("id", client.clientId);
            $card.attr("data-name", client.name.toLowerCase());
            $card.attr("data-address", client.address.toLowerCase());
            $card.attr("data-active", client.isConnected ? "1" : "0");
            $card.attr("data-handshake", client.lastHandshake || 0);
            $card.attr("data-traffic", (client.transferRx || 0) + (client.transferTx || 0));

            $("#clientList").append($card);
          });

          sortClientList();
          return;
        }

        const $listEmptyDefault = buildDefaultListItem(
          "No clients configured.",
        );
        $("#clientList").append($listEmptyDefault);
      },
      error: function (xhr, status, error) {
        // Only show modal if NOT a background poll
        if (!options.silent) {
          defaultModalError("Load clients", xhr, status);
        }
        // Log error for debugging
        console.error("Failed to load clients:", error);
      },
    });
  }

  // dynamic search
  // input instead of keyup because keyup does not work if x in the search field is pressed to clear it
  $("#searchClient").on("input", function () {
    const query = $(this).val().toLowerCase();
    let visibleCount = 0;

    $("#clientList li").each(function () {
      // ensure both values are always treated as strings - otherwise numbers will break the search bc of the value interpretation
      const name = String($(this).data("name") || "").toLowerCase();
      const address = String($(this).data("address") || "").toLowerCase();

      if (name.includes(query) || address.includes(query)) {
        $(this).show();
        visibleCount++;
      } else {
        $(this).hide();
      }
    });

    // remove any existing "No clients found" or "No clients configured" messages
    $("#clientList .no-items-mt").remove();

    if (query === "") {
      // if search is cleared and there are no clients, restore the default message
      if ($("#clientList li:visible").length === 0) {
        const $listEmptyDefault = buildDefaultListItem(
          "No clients configured.",
        );
        $("#clientList").append($listEmptyDefault);
      }
    } else if (visibleCount === 0) {
      // ff no items match the search, show "No clients found" message
      const $listEmptySearch = buildDefaultListItem("No clients found.");
      $("#clientList").append($listEmptySearch);
    }
  });

  $("#sortClient").on("change", function () {
    localStorage.setItem("clientSort", $(this).val());
    sortClientList();
  });

  $("#create-client").on("click", function (event) {
    event.preventDefault();

    var data = {};
    data["serverId"] = $("#serverSelect").val();
    $.ajax({
      url: "/api/v1/set-create-client-session",
      type: "POST",
      contentType: "application/json",
      data: JSON.stringify(data),
      success: function (response) {
        location.replace("/create-client");
      },
      error: function (xhr, status, error) {
        defaultModalError("Create client", xhr, status);
      },
    });
  });

  // get all available server options
  var optionValues = [];
  $("#serverSelect option").each(function () {
    optionValues.push($(this).val());
  });

  var optionCount = optionValues.length;
  if (optionCount === 0) {
    showModal(
      "Clients",
      "There are no servers configured. Please create a server",
      {
        showCloseButton: false,
        showActionButton: true,
        status: "error",
        onAction: function () {
          hideModal();
          location.replace("/create-server");
        },
      },
    );
  } else {
    // Initial load - show errors
    loadServerClients($("#serverSelect").val());
    // Polling - silent failures
    setInterval(function () {
      loadServerClients($("#serverSelect").val(), { silent: true });
    }, 5000);
    $("#serverSelect").change(function () {
      loadServerClients($(this).val());
    });
  }
});
