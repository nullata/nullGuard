/*
 * Copyright (c) 2026 nullata
 * SPDX-License-Identifier: Elastic-2.0
 * License: https://www.elastic.co/licensing/elastic-license
 */

// apply theme immediately to prevent flash
(function () {
  const theme = localStorage.getItem("theme") || "light";
  document.documentElement.setAttribute("data-theme", theme);
})();
