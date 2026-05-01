/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

function showModal(title, bodyContent, options = {}) {
  if (!title) {
    title = "Info";
  }
  $("#modal .modal-title").text(title);
  $("#modal .modal-body").html(bodyContent);

  var modalDialog = $("#modal .modal-dialog");
  modalDialog.removeClass("modal-cheatsheet");
  if (options.modalClass) {
    modalDialog.addClass(options.modalClass);
  }

  // remove any existing background color classes from the modal-header
  $("#modal .modal-header").removeClass(
    "bg-primary bg-info bg-success bg-danger bg-warning bg-secondary",
  );

  // set the background color based on the provided options
  if (options.status) {
    if (options.status === "success") {
      $("#modal .modal-header").addClass("bg-primary");
    } else if (options.status === "error") {
      $("#modal .modal-header").addClass("bg-danger");
    } else {
      $("#modal .modal-header").addClass("bg-secondary");
    }
  }

  var footer = $("#modal .modal-footer");
  footer.empty();

  // create and append the close button
  if (options.showCloseButton) {
    var closeIcon = $("<i>").addClass("fas fa-times me-2");
    var closeButton = $("<button>")
      .attr("type", "button")
      .addClass("btn btn-secondary")
      .attr("data-bs-dismiss", "modal")
      .append(closeIcon)
      .append("Close");

    footer.append(closeButton);
  }

  // create and append the action button
  if (options.showActionButton) {
    // determine action button name
    var actionName = "OK";
    if (options.actionName) {
      actionName = options.actionName;
    }

    var actionStyle = "btn btn-primary";
    if (options.actionStyle) {
      actionStyle = options.actionStyle;
    }

    // determine icon based on action name or style
    var iconClass = "fas fa-check";
    var actionNameLower = actionName.toLowerCase();

    if (
      actionNameLower.includes("delete") ||
      actionNameLower.includes("revoke") ||
      actionNameLower.includes("remove")
    ) {
      iconClass = "fas fa-trash";
    } else if (
      actionNameLower.includes("save") ||
      actionNameLower.includes("saved")
    ) {
      iconClass = "fas fa-check";
    } else if (actionStyle.includes("danger")) {
      iconClass = "fas fa-exclamation-triangle";
    }

    var actionIcon = $("<i>").addClass(iconClass + " me-2");
    var actionButton = $("<button>")
      .attr("type", "button")
      .addClass(actionStyle)
      .attr("id", "actionButton")
      .append(actionIcon)
      .append(actionName);

    // attach click handler during creation if provided
    if (options.onAction) {
      actionButton.on("click", options.onAction);
    }

    footer.append(actionButton);
  }

  // show the modal and configure it not to close when clicking outside or pressing escape
  var modalEl = document.getElementById("modal");
  var existingModal = bootstrap.Modal.getInstance(modalEl);
  if (existingModal) {
    existingModal.dispose();
  }
  var modal = new bootstrap.Modal(modalEl, {
    backdrop: options.backdrop || "static", // prevents closing on outside click
    keyboard: options.keyboard || false, // prevents closing on pressing escape
  });
  modal.show();

  // if provided, add event listener for the close button click
  if (options.onClose) {
    $("#modal").on("hidden.bs.modal", function () {
      options.onClose();
      $(this).off("hidden.bs.modal"); // clean up event listener
    });
  }
}

// Helper function to hide the modal
function hideModal() {
  var modal = bootstrap.Modal.getInstance(document.getElementById('modal'));
  if (modal) modal.hide();
}

function defaultModalSuccess(title, response) {
  showModal(title, response.message, {
    showCloseButton: false,
    showActionButton: true,
    status: response.status,
    onAction: function () {
      hideModal();
      location.reload();
    },
  });
}

function defaultModalError(title, xhr, status) {
  var responseMessage;
  try {
    responseMessage = JSON.parse(xhr.responseText).message;
  } catch (e) {
    responseMessage = "An unknown error occurred";
  }
  showModal(title, responseMessage, {
    showCloseButton: true,
    status: status,
    onClose: function () {
      hideModal();
    },
  });
}

function normalize(str) {
  // ensure values are treated as strings and common whitespace chars are normalized
  return String(str || "")
    .trim()
    .replace(/\r\n|\r/g, "\n");
}

// theme switching gunctionality
(function () {
  // get the current theme from localstorage, default to 'light'
  function getTheme() {
    return localStorage.getItem("theme") || "light";
  }

  // set the theme in localstorage and apply it to the document
  function setTheme(theme) {
    localStorage.setItem("theme", theme);
    document.documentElement.setAttribute("data-theme", theme);
    updateThemeIcon(theme);
  }

  // update theme toggle icon
  function updateThemeIcon(theme) {
    const icon = document.getElementById("theme-icon");
    if (icon) {
      if (theme === "dark") {
        icon.className = "fas fa-sun";
      } else {
        icon.className = "fas fa-moon";
      }
    }
  }

  // toggle between light and dark themes
  function toggleTheme() {
    const currentTheme = getTheme();
    const newTheme = currentTheme === "dark" ? "light" : "dark";
    setTheme(newTheme);
  }

  // initialize theme on page load
  function initTheme() {
    const savedTheme = getTheme();
    setTheme(savedTheme);
  }

  // set up event listeners when dom is ready
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function () {
      initTheme();
      const toggleButton = document.getElementById("theme-toggle");
      if (toggleButton) {
        toggleButton.addEventListener("click", toggleTheme);
      }
    });
  } else {
    // dom already loaded
    initTheme();
    const toggleButton = document.getElementById("theme-toggle");
    if (toggleButton) {
      toggleButton.addEventListener("click", toggleTheme);
    }
  }
})();
