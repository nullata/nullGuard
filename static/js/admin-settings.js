/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

//store loaded tokens for validation
let existingTokens = [];

document.addEventListener("DOMContentLoaded", function () {
  loadTokens();

  document
    .getElementById("create-token-form")
    .addEventListener("submit", function (e) {
      e.preventDefault();
      createToken();
    });

  document
    .getElementById("change-password-form")
    .addEventListener("submit", function (e) {
      e.preventDefault();
      changePassword();
    });
});

// ===== API TOKENS =====

function createToken() {
  const name = document.getElementById("tokenName").value.trim();
  const expiresInDays =
    parseInt(document.getElementById("expiresInDays").value) || 0;

  // Check for duplicate token name
  const isDuplicate = existingTokens.some(
    (token) => token.name.toLowerCase() === name.toLowerCase(),
  );

  if (isDuplicate) {
    showModal(
      "Duplicate Token Name",
      `A token with the name "${escapeHtml(name)}" already exists. Please choose a different name.`,
      {
        showCloseButton: true,
        showActionButton: false,
        status: "error",
      },
    );
    return;
  }

  fetch("/api/v1/tokens", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      name: name,
      expires_in_days: expiresInDays,
    }),
  })
    .then((response) => response.json())
    .then((data) => {
      if (data.status === "success") {
        showTokenModal(data.data.token);
        loadTokens();
        document.getElementById("create-token-form").reset();
      } else {
        showModal("Error", data.message || "Failed to create token", {
          showActionButton: true,
          status: "error",
        });
      }
    })
    .catch((error) => {
      console.error("Error:", error);
      showModal("Error", "Failed to create token", {
        showActionButton: true,
        status: "error",
      });
    });
}

function showTokenModal(token) {
  const modalBody = `
    <div class="">
      <h5>Token Created Successfully!</h5>
      <p><strong>Important:</strong> Save this token now. You won't be able to see it again!</p>
      <div class="input-group mb-3">
        <input type="text" class="form-control" id="modalTokenValue" value="${escapeHtml(token)}" readonly>
        <button class="btn btn-secondary" type="button" id="modalCopyTokenBtn"><i class="fas fa-copy me-1"></i>Copy</button>
      </div>
    </div>
  `;

  showModal("API Token Created", modalBody, {
    showCloseButton: false,
    showActionButton: true,
    actionName: "I've saved the token",
    backdrop: "static",
    keyboard: false,
    onAction: function () {
      hideModal();
    },
  });

  setTimeout(() => {
    const copyBtn = document.getElementById("modalCopyTokenBtn");
    if (copyBtn) {
      copyBtn.addEventListener("click", function () {
        const tokenInput = document.getElementById("modalTokenValue");
        const text = tokenInput.value;

        function onCopied() {
          const originalHTML = copyBtn.innerHTML;
          copyBtn.innerHTML = '<i class="fas fa-check me-1"></i>Copied!';
          setTimeout(() => {
            copyBtn.innerHTML = originalHTML;
          }, 2000);
        }

        // navigator.clipboard is only available in secure contexts (https/localhost)
        // fall back to execCommand("copy") for plain http
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(text).then(onCopied);
        } else {
          tokenInput.select();
          document.execCommand("copy");
          onCopied();
        }
      });
    }
  }, 100);
}

function loadTokens() {
  fetch("/api/v1/tokens", {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
    },
  })
    .then((response) => response.json())
    .then((data) => {
      if (data.status === "success") {
        existingTokens = data.data || [];
        displayTokens(existingTokens);
      } else {
        document.getElementById("tokens-container").innerHTML =
          '<p class="text-danger">Failed to load tokens</p>';
      }
    })
    .catch((error) => {
      console.error("Error:", error);
      document.getElementById("tokens-container").innerHTML =
        '<p class="text-danger">Failed to load tokens</p>';
    });
}

function displayTokens(tokens) {
  const container = document.getElementById("tokens-container");

  if (tokens.length === 0) {
    container.innerHTML =
      '<p class="text-muted">No API tokens found. Create one above to get started.</p>';
    return;
  }

  let html =
    '<div class="table-responsive"><table class="table table-striped table-hover">';
  html += "<thead><tr>";
  html += "<th>Name</th>";
  html += "<th>Created</th>";
  html += "<th>Expires</th>";
  html += "<th>Last Used</th>";
  html += "<th>Actions</th>";
  html += "</tr></thead><tbody>";

  tokens.forEach((token) => {
    const createdDate = new Date(token.created_at);
    const expiresDate = token.expires_at ? new Date(token.expires_at) : null;
    const lastUsedDate = token.last_used_at
      ? new Date(token.last_used_at)
      : null;

    html += "<tr>";
    html += `<td><strong>${escapeHtml(token.name)}</strong>`;
    if (token.created_by_ip) {
      html += `<br><small class="text-muted">Created from: ${escapeHtml(token.created_by_ip)}</small>`;
    }
    html += "</td>";
    html += `<td>${formatDateTime(createdDate)}</td>`;
    html += `<td>${expiresDate ? formatDateTime(expiresDate) : "Never"}</td>`;
    html += `<td>${lastUsedDate ? formatDateTime(lastUsedDate) : "Never"}`;
    if (token.last_used_ip) {
      html += `<br><small class="text-muted">From: ${escapeHtml(token.last_used_ip)}</small>`;
    }
    html += "</td>";
    html += `<td><button class="btn btn-sm btn-danger" onclick="revokeToken(${token.id}, '${escapeHtml(token.name)}')"><i class="fas fa-trash me-1"></i>Revoke</button></td>`;
    html += "</tr>";
  });

  html += "</tbody></table></div>";
  container.innerHTML = html;
}

function revokeToken(tokenId, tokenName) {
  const modalBody = `
    <p>Are you sure you want to revoke the token <strong>"${escapeHtml(tokenName)}"</strong>?</p>
    <p class="text-danger">This action cannot be undone.</p>
  `;

  showModal("Revoke Token", modalBody, {
    showCloseButton: true,
    showActionButton: true,
    actionName: "Revoke Token",
    actionStyle: "btn btn-danger",
    status: "error",
    onAction: function () {
      performRevokeToken(tokenId);
    },
  });
}

function performRevokeToken(tokenId) {
  fetch(`/api/v1/tokens/${tokenId}`, {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
    },
  })
    .then((response) => response.json())
    .then((data) => {
      if (data.status === "success") {
        hideModal();
        loadTokens();
      } else {
        showModal("Error", data.message || "Failed to revoke token", {
          showActionButton: true,
          status: "error",
        });
      }
    })
    .catch((error) => {
      console.error("Error:", error);
      showModal("Error", "Failed to revoke token", {
        showActionButton: true,
        status: "error",
      });
    });
}

// ===== PASSWORD CHANGE =====

function changePassword() {
  const formData = {
    old_password: document.getElementById("old_password").value,
    new_password: document.getElementById("new_password").value,
    new_password_confirm: document.getElementById("new_password_confirm").value,
    new_password_hint:
      document.getElementById("new_password_hint").value.trim() || null,
  };

  if (formData.new_password !== formData.new_password_confirm) {
    showModal("Error", "New passwords do not match", {
      showActionButton: false,
      showCloseButton: true,
      status: "error",
    });
    return;
  }

  fetch("/api/v1/change-password", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(formData),
  })
    .then((response) => response.json())
    .then((data) => {
      if (data.status === "success") {
        document.getElementById("change-password-form").reset();
        showModal("Success", data.message || "Password changed successfully", {
          showActionButton: false,
          showCloseButton: true,
          status: "success",
        });
      } else {
        showModal("Error", data.message || "Failed to change password", {
          showActionButton: false,
          showCloseButton: true,
          status: "error",
        });
      }
    })
    .catch((error) => {
      console.error("Error:", error);
      showModal("Error", "Failed to change password", {
        showActionButton: false,
        showCloseButton: true,
        status: "error",
      });
    });
}

// ===== HELPERS =====

function formatDateTime(date) {
  return date.toLocaleString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function escapeHtml(text) {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}
