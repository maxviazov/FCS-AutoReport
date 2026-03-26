(function () {
  const logEl = document.getElementById("log");
  const rawPath = document.getElementById("rawPath");
  const templatePath = document.getElementById("templatePath");
  const outputDir = document.getElementById("outputDir");
  const btnGenerate = document.getElementById("btnGenerate");

  function tr(key) { return window.i18n && window.i18n.t ? window.i18n.t(key) : key; }
  function trParam(key, params) { return window.i18n && window.i18n.tParam ? window.i18n.tParam(key, params) : key; }

  function filterTableBySearch(tbodyId, searchInputId) {
    var tbody = document.getElementById(tbodyId);
    var input = document.getElementById(searchInputId);
    if (!tbody || !input) return;
    var q = (input.value || "").trim().toLowerCase();
    var rows = tbody.querySelectorAll("tr");
    for (var i = 0; i < rows.length; i++) {
      var row = rows[i];
      if (q === "") {
        row.style.display = "";
      } else {
        var text = (row.textContent || "").toLowerCase();
        row.style.display = text.indexOf(q) !== -1 ? "" : "none";
      }
    }
  }

  function log(msg, type) {
    const line = document.createElement("span");
    line.className = type || "";
    line.textContent = "[" + new Date().toLocaleTimeString() + "] " + msg + "\n";
    logEl.appendChild(line);
    logEl.scrollTop = logEl.scrollHeight;
  }

  function getBackend() {
    if (window.go && window.go.app && window.go.app.WailsApp) return window.go.app.WailsApp;
    if (window.Go && typeof window.Go.app !== "undefined" && window.Go.app.WailsApp) return window.Go.app.WailsApp;
    if (typeof window.GenerateReport === "function") return { GenerateReport: window.GenerateReport };
    return null;
  }

  document.querySelectorAll(".lang-btn").forEach(function (btn) {
    btn.addEventListener("click", function () {
      var lang = btn.getAttribute("data-lang");
      if (window.i18n && window.i18n.setLang) window.i18n.setLang(lang);
    });
  });

  document.getElementById("btnRaw").addEventListener("click", async function () {
    const backend = getBackend();
    if (!backend || typeof backend.SelectRawReport !== "function") {
      log(tr("msg_backendUnavailable"), "error");
      return;
    }
    try {
      const path = await backend.SelectRawReport();
      if (path) {
        rawPath.value = path;
        log(tr("reportGen_rawLabel").replace(" (משקל.xlsx)", "") + ": " + path, "success");
      }
    } catch (e) {
      log((e && e.message ? e.message : String(e)), "error");
    }
  });

  document.getElementById("btnTemplate").addEventListener("click", async function () {
    const backend = getBackend();
    if (!backend || typeof backend.SelectTemplate !== "function") {
      log(tr("msg_backendUnavailable"), "error");
      return;
    }
    try {
      const path = await backend.SelectTemplate();
      if (path) {
        templatePath.value = path;
        log(tr("reportGen_templateLabel") + ": " + path, "success");
      }
    } catch (e) {
      log((e && e.message ? e.message : String(e)), "error");
    }
  });

  document.getElementById("btnOutput").addEventListener("click", async function () {
    const backend = getBackend();
    if (!backend || typeof backend.SelectOutputDir !== "function") {
      log(tr("msg_backendUnavailable"), "error");
      return;
    }
    try {
      const path = await backend.SelectOutputDir();
      if (path) {
        outputDir.value = path;
        log(tr("reportGen_outputLabel") + ": " + path, "success");
      }
    } catch (e) {
      log((e && e.message ? e.message : String(e)), "error");
    }
  });

  document.getElementById("btnImportCities").addEventListener("click", async function () {
    const backend = getBackend();
    if (!backend || typeof backend.ImportCitiesDict !== "function") {
      log(tr("msg_backendUnavailable"), "error");
      return;
    }
    log(tr("msg_selectCities"));
    try {
      const msg = await backend.ImportCitiesDict();
      if (msg) log(msg, "success");
    } catch (e) {
      log((e && e.message ? e.message : String(e)), "error");
    }
  });

  document.getElementById("btnImportDrivers").addEventListener("click", async function () {
    const backend = getBackend();
    if (!backend || typeof backend.ImportDriversDict !== "function") {
      log(tr("msg_backendUnavailable"), "error");
      return;
    }
    log(tr("msg_selectDrivers"));
    try {
      const msg = await backend.ImportDriversDict();
      if (msg) log(msg, "success");
    } catch (e) {
      log((e && e.message ? e.message : String(e)), "error");
    }
  });

  document.getElementById("btnImportItems").addEventListener("click", async function () {
    const backend = getBackend();
    if (!backend || typeof backend.ImportItemsDict !== "function") {
      log(tr("msg_backendUnavailable"), "error");
      return;
    }
    log(tr("msg_selectItems"));
    try {
      const msg = await backend.ImportItemsDict();
      if (msg) log(msg, "success");
    } catch (e) {
      log((e && e.message ? e.message : String(e)), "error");
    }
  });

  const btnGenerateEl = btnGenerate;
  const generateStatusEl = document.getElementById("generateStatus");
  const btnOpenReportFolder = document.getElementById("btnOpenReportFolder");
  var lastSavedReportPath = "";

  function setLastReportPath(path) {
    lastSavedReportPath = path || "";
    if (btnOpenReportFolder) btnOpenReportFolder.style.display = lastSavedReportPath ? "inline-block" : "none";
  }

  btnGenerate.addEventListener("click", async function () {
    const raw = rawPath.value.trim();
    const template = templatePath.value.trim();
    const output = outputDir.value.trim();
    if (!raw || !template || !output) {
      log(tr("msg_fillPaths"), "error");
      if (generateStatusEl) {
        generateStatusEl.textContent = tr("msg_fillPathsShort");
        generateStatusEl.className = "generate-status error";
      }
      return;
    }

    const backend = getBackend();
    if (!backend || typeof backend.GenerateReport !== "function") {
      log(tr("msg_backendUnavailable"), "error");
      if (generateStatusEl) {
        generateStatusEl.textContent = tr("msg_backendUnavailable").split(".")[0];
        generateStatusEl.className = "generate-status error";
      }
      return;
    }

    btnGenerateEl.disabled = true;
    if (generateStatusEl) {
      generateStatusEl.textContent = tr("msg_generating");
      generateStatusEl.className = "generate-status";
    }
    log(tr("msg_generating"));
    try {
      const savedPath = await backend.GenerateReport(raw, template, output);
      log(tr("msg_done") + ": " + savedPath, "success");
      log(tr("msg_savedPaths"));
      setLastReportPath(savedPath);
      if (generateStatusEl) {
        generateStatusEl.textContent = tr("msg_done") + ": " + savedPath;
        generateStatusEl.className = "generate-status";
      }
      if (typeof backend.OpenFileLocation === "function") {
        try {
          await backend.OpenFileLocation(savedPath);
        } catch (e) {
          log((e && e.message ? e.message : String(e)), "error");
        }
      }
    } catch (err) {
      const errMsg = err && err.message ? err.message : String(err);
      if (errMsg.indexOf("unresolved_cities") !== -1) {
        log(tr("msg_unresolvedNoSave"), "error");
        if (generateStatusEl) {
          generateStatusEl.textContent = tr("msg_unresolvedNoSaveShort");
          generateStatusEl.className = "generate-status error";
        }
        var unresolved = [];
        if (typeof backend.GetLastUnresolvedCities === "function") {
          try {
            var list = await backend.GetLastUnresolvedCities();
            if (list && list.length) unresolved = list;
          } catch (e) {}
        }
        showUnresolvedModal(unresolved, backend);
      } else {
        log(errMsg, "error");
        if (generateStatusEl) {
          generateStatusEl.textContent = errMsg;
          generateStatusEl.className = "generate-status error";
        }
      }
    } finally {
      btnGenerateEl.disabled = false;
    }
  });

  function removeUnresolvedRow(rowEl) {
    if (rowEl && rowEl.parentNode) rowEl.parentNode.removeChild(rowEl);
    var list = document.getElementById("unresolvedList");
    if (list && list.children.length === 0) {
      var modal = document.getElementById("unresolvedModal");
      if (modal) modal.style.display = "none";
    }
  }

  async function showUnresolvedModal(unresolved, backend) {
    var modal = document.getElementById("unresolvedModal");
    var listEl = document.getElementById("unresolvedList");
    var retryStatus = document.getElementById("unresolvedRetryStatus");
    if (!modal || !listEl) return;
    if (retryStatus) {
      retryStatus.textContent = "";
      retryStatus.className = "unresolved-retry-status";
    }
    listEl.innerHTML = "";
    if (!unresolved || unresolved.length === 0) {
      var hint = document.createElement("p");
      hint.className = "dict-desc";
      hint.style.marginBottom = "1rem";
      hint.textContent = tr("msg_unresolvedHint") || "Добавьте города/алиасы во вкладке «Города» и нажмите «Повторить генерацию».";
      listEl.appendChild(hint);
    }
    var cities = [];
    if (typeof backend.GetCities === "function") {
      try {
        cities = await backend.GetCities();
      } catch (e) {}
    }
    (unresolved || []).forEach(function (name) {
      var row = document.createElement("div");
      row.className = "unresolved-row";
      row.setAttribute("data-name", name);
      var nameSpan = document.createElement("span");
      nameSpan.className = "unresolved-name";
      nameSpan.textContent = name;
      row.appendChild(nameSpan);

      var sel = document.createElement("select");
      sel.innerHTML = "<option value=\"\">" + (tr("unresolved_resolve") || "—") + "</option>";
      cities.forEach(function (c) {
        var opt = document.createElement("option");
        opt.value = String(c.id != null ? c.id : c.ID);
        opt.textContent = (c.name || c.Name || "") + " (" + (c.code || c.Code || "") + ")";
        sel.appendChild(opt);
      });
      row.appendChild(sel);

      var btnAlias = document.createElement("button");
      btnAlias.type = "button";
      btnAlias.className = "btn-secondary btn-small";
      btnAlias.textContent = tr("unresolved_addAlias");
      btnAlias.addEventListener("click", async function () {
        var cityId = parseInt(sel.value, 10);
        if (!cityId || !backend.AddCityAlias) return;
        try {
          await backend.AddCityAlias(cityId, name);
          log(tr("msg_citySaved") + " (алиас: " + name + ")", "success");
          removeUnresolvedRow(row);
        } catch (e) {
          log((e && e.message ? e.message : String(e)), "error");
        }
      });
      row.appendChild(btnAlias);

      var codeInput = document.createElement("input");
      codeInput.type = "text";
      codeInput.className = "unresolved-code";
      codeInput.placeholder = tr("unresolved_newCity") || "N126";
      row.appendChild(codeInput);

      var btnNew = document.createElement("button");
      btnNew.type = "button";
      btnNew.className = "btn-secondary btn-small";
      btnNew.textContent = tr("unresolved_create");
      btnNew.addEventListener("click", async function () {
        var code = (codeInput.value || "").trim();
        if (!code || !backend.SaveCity) return;
        try {
          await backend.SaveCity({ id: 0, name: name, code: code, aliases: [] });
          log(tr("msg_citySaved") + " (" + name + " → " + code + ")", "success");
          removeUnresolvedRow(row);
        } catch (e) {
          log((e && e.message ? e.message : String(e)), "error");
        }
      });
      row.appendChild(btnNew);

      listEl.appendChild(row);
    });
    modal.style.display = "flex";
  }

  document.getElementById("btnUnresolvedDone").addEventListener("click", function () {
    var modal = document.getElementById("unresolvedModal");
    if (modal) modal.style.display = "none";
  });

  document.getElementById("btnUnresolvedRetry").addEventListener("click", async function () {
    var statusEl = document.getElementById("unresolvedRetryStatus");
    var setStatus = function (msg, isError) {
      if (!statusEl) return;
      statusEl.textContent = msg || "";
      statusEl.className = "unresolved-retry-status" + (isError ? " error" : "");
    };
    var backend = getBackend();
    if (!backend || typeof backend.GenerateReport !== "function") {
      var m0 = tr("unresolved_retry_noBackend");
      setStatus(m0, true);
      log(m0, "error");
      return;
    }
    var raw = (document.getElementById("rawPath") && document.getElementById("rawPath").value) ? document.getElementById("rawPath").value.trim() : "";
    var template = (document.getElementById("templatePath") && document.getElementById("templatePath").value) ? document.getElementById("templatePath").value.trim() : "";
    var output = (document.getElementById("outputDir") && document.getElementById("outputDir").value) ? document.getElementById("outputDir").value.trim() : "";
    if (!raw || !template || !output) {
      setStatus(tr("unresolved_retry_fillPaths"), true);
      log(tr("msg_fillPathsShort"), "error");
      return;
    }
    var btn = document.getElementById("btnUnresolvedRetry");
    if (btn) btn.disabled = true;
    setStatus(tr("unresolved_retry_generating"), false);
    try {
      var savedPath = await backend.GenerateReport(raw, template, output);
      setStatus("", false);
      log(tr("msg_done") + ": " + savedPath, "success");
      setLastReportPath(savedPath);
      if (generateStatusEl) {
        generateStatusEl.textContent = tr("msg_done") + ": " + savedPath;
        generateStatusEl.className = "generate-status";
      }
      if (typeof backend.OpenFileLocation === "function") {
        try { await backend.OpenFileLocation(savedPath); } catch (e) {}
      }
      var modal = document.getElementById("unresolvedModal");
      if (modal) modal.style.display = "none";
    } catch (err) {
      var errMsg = err && err.message ? err.message : String(err);
      setStatus(errMsg, true);
      log(errMsg, "error");
      if (errMsg.indexOf("unresolved_cities") !== -1 && typeof backend.GetLastUnresolvedCities === "function") {
        try {
          var unresolved = await backend.GetLastUnresolvedCities();
          showUnresolvedModal(unresolved || [], backend);
        } catch (e) {}
      }
    } finally {
      if (btn) btn.disabled = false;
    }
  });

  document.getElementById("unresolvedModal").addEventListener("click", function (e) {
    if (e.target.id === "unresolvedModal") e.target.style.display = "none";
  });

  async function loadInitialSettings() {
    const backend = getBackend();
    if (!backend || typeof backend.GetSettings !== "function") return;
    try {
      const s = await backend.GetSettings();
      const raw = (s && (s.inputFolder ?? s.InputFolder ?? ""));
      const out = (s && (s.outputFolder ?? s.OutputFolder ?? ""));
      const tpl = (s && (s.templatePath ?? s.TemplatePath ?? ""));
      const smtpHost = (s && (s.smtpHost ?? s.SMTPHost ?? ""));
      const smtpPort = (s && (s.smtpPort ?? s.SMTPPort ?? 587));
      const smtpUser = (s && (s.smtpUser ?? s.SMTPUser ?? ""));
      const smtpPassword = (s && (s.smtpPassword ?? s.SMTPPassword ?? ""));
      const imapHost = (s && (s.imapHost ?? s.IMAPHost ?? ""));
      const imapPort = (s && (s.imapPort ?? s.IMAPPort ?? 993));
      const imapUser = (s && (s.imapUser ?? s.IMAPUser ?? ""));
      const imapPassword = (s && (s.imapPassword ?? s.IMAPPassword ?? ""));
      const autoSend = !!(s && (s.autoSend ?? s.AutoSend));
      const watchEnabled = !!(s && (s.watchEnabled ?? s.WatchEnabled));
      const watchFolder = (s && (s.watchFolder ?? s.WatchFolder ?? ""));
      if (raw || out || tpl) {
        rawPath.value = raw;
        outputDir.value = out;
        templatePath.value = tpl;
        log(tr("msg_savedPathsLoaded"), "success");
      }
      var el;
      el = document.getElementById("smtpHost"); if (el) el.value = smtpHost;
      el = document.getElementById("smtpPort"); if (el) el.value = String(smtpPort || 587);
      el = document.getElementById("smtpUser"); if (el) el.value = smtpUser;
      el = document.getElementById("smtpPassword"); if (el) el.value = smtpPassword;
      el = document.getElementById("imapHost"); if (el) el.value = imapHost;
      el = document.getElementById("imapPort"); if (el) el.value = String(imapPort || 993);
      el = document.getElementById("imapUser"); if (el) el.value = imapUser;
      el = document.getElementById("imapPassword"); if (el) el.value = imapPassword;
      el = document.getElementById("autoSendEnabled"); if (el) el.checked = autoSend;
      el = document.getElementById("watchEnabled"); if (el) el.checked = watchEnabled;
      el = document.getElementById("watchFolder"); if (el) el.value = watchFolder || raw || out;
    } catch (e) {}
  }

  async function loadCitiesTable() {
    const backend = getBackend();
    if (!backend || typeof backend.GetCities !== "function") return;
    const tbody = document.getElementById("citiesTbody");
    tbody.innerHTML = "";
    try {
      const cities = await backend.GetCities();
      if (!cities || !cities.length) {
        tbody.innerHTML = "<tr><td colspan=\"5\">" + tr("noData_cities") + "</td></tr>";
        return;
      }
      const aliasesStr = function (arr) {
        if (!arr || !arr.length) return "";
        return Array.isArray(arr) ? arr.join(", ") : String(arr);
      };
      cities.forEach(function (c) {
        const id = c.id ?? c.ID ?? 0;
        const name = c.name ?? c.Name ?? "";
        const code = c.code ?? c.Code ?? "";
        const aliases = c.aliases ?? c.Aliases ?? [];
        const row = document.createElement("tr");
        row.innerHTML =
          "<td>" + (id || "") + "</td>" +
          "<td>" + (name || "") + "</td>" +
          "<td>" + (code || "") + "</td>" +
          "<td>" + aliasesStr(aliases) + "</td>" +
          "<td class=\"btn-cell\">" +
          "<button type=\"button\" class=\"btn-secondary btn-edit-city\" data-id=\"" + (id || 0) + "\" data-name=\"" + (name || "").replace(/"/g, "&quot;") + "\" data-code=\"" + (code || "").replace(/"/g, "&quot;") + "\" data-aliases=\"" + (aliasesStr(aliases) || "").replace(/"/g, "&quot;") + "\">" + tr("btn_edit") + "</button> " +
          "<button type=\"button\" class=\"btn-secondary btn-delete-city\" data-id=\"" + (id || 0) + "\">" + tr("btn_delete") + "</button>" +
          "</td>";
        tbody.appendChild(row);
      });
      tbody.querySelectorAll(".btn-edit-city").forEach(function (btn) {
        btn.addEventListener("click", function () {
          document.getElementById("cityId").value = btn.getAttribute("data-id");
          document.getElementById("cityName").value = btn.getAttribute("data-name") || "";
          document.getElementById("cityCode").value = btn.getAttribute("data-code") || "";
          document.getElementById("cityAliases").value = btn.getAttribute("data-aliases") || "";
          document.getElementById("cityModalTitle").textContent = tr("city_editTitle");
          document.getElementById("cityModal").style.display = "flex";
        });
      });
      tbody.querySelectorAll(".btn-delete-city").forEach(function (btn) {
        btn.addEventListener("click", function () {
          var id = parseInt(btn.getAttribute("data-id"), 10);
          if (!confirm(trParam("msg_confirmDeleteCity", { id: id }))) return;
          const b = getBackend();
          if (!b || !b.DeleteCity) return;
          b.DeleteCity(id).then(function () {
            log(tr("msg_cityDeleted"), "success");
            loadCitiesTable();
          }).catch(function (e) {
            log((e && e.message ? e.message : String(e)), "error");
          });
        });
      });
      filterTableBySearch("citiesTbody", "searchCities");
    } catch (e) {
      log((e && e.message ? e.message : String(e)), "error");
    }
  }

  document.getElementById("btnRefreshCities").addEventListener("click", function () {
    loadCitiesTable();
    log(tr("msg_listRefreshed"));
  });

  document.getElementById("btnAddCity").addEventListener("click", function () {
    document.getElementById("cityId").value = "0";
    document.getElementById("cityName").value = "";
    document.getElementById("cityCode").value = "";
    document.getElementById("cityAliases").value = "";
    document.getElementById("cityModalTitle").textContent = tr("city_addTitle");
    document.getElementById("cityModal").style.display = "flex";
  });

  document.getElementById("btnSaveCity").addEventListener("click", async function () {
    const backend = getBackend();
    if (!backend || typeof backend.SaveCity !== "function") {
      log(tr("msg_backendUnavailable"), "error");
      return;
    }
    var id = parseInt(document.getElementById("cityId").value, 10) || 0;
    var name = document.getElementById("cityName").value.trim();
    var code = document.getElementById("cityCode").value.trim();
    var aliasesStr = document.getElementById("cityAliases").value.trim();
    var aliases = aliasesStr ? aliasesStr.split(",").map(function (s) { return s.trim(); }).filter(Boolean) : [];
    if (!name) {
      log(tr("msg_enterCityName"), "error");
      return;
    }
    var city = { ID: id, Name: name, Code: code, Aliases: aliases };
    try {
      await backend.SaveCity(city);
      log(tr("msg_citySaved"), "success");
      document.getElementById("cityModal").style.display = "none";
      loadCitiesTable();
    } catch (e) {
      log("Ошибка сохранения: " + (e && e.message ? e.message : String(e)), "error");
    }
  });

  document.getElementById("btnCancelCity").addEventListener("click", function () {
    document.getElementById("cityModal").style.display = "none";
  });

  document.getElementById("cityModal").addEventListener("click", function (e) {
    if (e.target === document.getElementById("cityModal")) {
      document.getElementById("cityModal").style.display = "none";
    }
  });

  function switchTab(tabName) {
    ["tabContentCities", "tabContentDrivers", "tabContentItems"].forEach(function (id) {
      document.getElementById(id).style.display = id === "tabContent" + tabName ? "block" : "none";
    });
    ["tabBtnCities", "tabBtnDrivers", "tabBtnItems"].forEach(function (id) {
      var btn = document.getElementById(id);
      btn.classList.toggle("active", id === "tabBtn" + tabName);
      btn.classList.toggle("tab-btn-active", id === "tabBtn" + tabName);
    });
  }

  var searchCitiesEl = document.getElementById("searchCities");
  if (searchCitiesEl) {
    searchCitiesEl.addEventListener("input", function () { filterTableBySearch("citiesTbody", "searchCities"); });
    searchCitiesEl.addEventListener("change", function () { filterTableBySearch("citiesTbody", "searchCities"); });
  }
  var searchDriversEl = document.getElementById("searchDrivers");
  if (searchDriversEl) {
    searchDriversEl.addEventListener("input", function () { filterTableBySearch("driversTbody", "searchDrivers"); });
    searchDriversEl.addEventListener("change", function () { filterTableBySearch("driversTbody", "searchDrivers"); });
  }

  document.getElementById("tabBtnCities").addEventListener("click", function () { switchTab("Cities"); });
  document.getElementById("tabBtnDrivers").addEventListener("click", function () {
    switchTab("Drivers");
    loadDriversTable();
  });
  document.getElementById("tabBtnItems").addEventListener("click", function () {
    switchTab("Items");
    loadItemsTable();
  });

  async function loadDriversTable() {
    const backend = getBackend();
    if (!backend || typeof backend.GetDrivers !== "function") return;
    const tbody = document.getElementById("driversTbody");
    tbody.innerHTML = "";
    try {
      const list = await backend.GetDrivers();
      if (!list || !list.length) {
        tbody.innerHTML = "<tr><td colspan=\"6\">" + tr("noData_drivers") + "</td></tr>";
        return;
      }
      list.forEach(function (d) {
        const agent = (d.agent_name ?? d.AgentName ?? "").replace(/"/g, "&quot;");
        const name = (d.driver_name ?? d.DriverName ?? "").replace(/"/g, "&quot;");
        const car = (d.car_number ?? d.CarNumber ?? "").replace(/"/g, "&quot;");
        const phone = (d.phone ?? d.Phone ?? "").replace(/"/g, "&quot;");
        const cities = (d.city_codes ?? d.CityCodes ?? "").replace(/"/g, "&quot;");
        const row = document.createElement("tr");
        row.innerHTML =
          "<td>" + (d.agent_name ?? d.AgentName ?? "") + "</td><td>" + (d.driver_name ?? d.DriverName ?? "") + "</td><td>" + (d.car_number ?? d.CarNumber ?? "") + "</td><td>" + (d.phone ?? d.Phone ?? "") + "</td><td>" + (d.city_codes ?? d.CityCodes ?? "") + "</td>" +
          "<td class=\"btn-cell\"><button type=\"button\" class=\"btn-secondary btn-edit-driver\" data-agent=\"" + agent + "\" data-name=\"" + name + "\" data-car=\"" + car + "\" data-phone=\"" + phone + "\" data-cities=\"" + cities + "\">" + tr("btn_edit") + "</button> " +
          "<button type=\"button\" class=\"btn-secondary btn-delete-driver\" data-agent=\"" + agent + "\">" + tr("btn_delete") + "</button></td>";
        tbody.appendChild(row);
      });
      tbody.querySelectorAll(".btn-edit-driver").forEach(function (btn) {
        btn.addEventListener("click", function () {
          document.getElementById("driverOriginalAgent").value = btn.getAttribute("data-agent") || "";
          document.getElementById("driverAgent").value = btn.getAttribute("data-agent") || "";
          document.getElementById("driverName").value = btn.getAttribute("data-name") || "";
          document.getElementById("driverCar").value = btn.getAttribute("data-car") || "";
          document.getElementById("driverPhone").value = btn.getAttribute("data-phone") || "";
          document.getElementById("driverCities").value = btn.getAttribute("data-cities") || "";
          document.getElementById("driverModalTitle").textContent = tr("driver_editTitle");
          document.getElementById("driverModal").style.display = "flex";
        });
      });
      tbody.querySelectorAll(".btn-delete-driver").forEach(function (btn) {
        btn.addEventListener("click", function () {
          var agent = btn.getAttribute("data-agent") || "";
          if (!confirm(trParam("msg_confirmDeleteDriver", { name: agent }))) return;
          var b = getBackend();
          if (!b || typeof b.DeleteDriver !== "function") {
            log(tr("msg_backendUnavailable"), "error");
            return;
          }
          b.DeleteDriver(agent).then(function () {
            log(tr("msg_driverDeleted"), "success");
            loadDriversTable();
          }).catch(function (e) {
            log("Ошибка удаления: " + (e && e.message ? e.message : String(e)), "error");
          });
        });
      });
      filterTableBySearch("driversTbody", "searchDrivers");
    } catch (e) {
      log((e && e.message ? e.message : String(e)), "error");
    }
  }

  document.getElementById("btnRefreshDrivers").addEventListener("click", function () {
    loadDriversTable();
    log(tr("msg_listRefreshed"));
  });
  document.getElementById("btnAddDriver").addEventListener("click", function () {
    document.getElementById("driverOriginalAgent").value = "";
    document.getElementById("driverAgent").value = "";
    document.getElementById("driverName").value = "";
    document.getElementById("driverCar").value = "";
    document.getElementById("driverPhone").value = "";
    document.getElementById("driverCities").value = "";
    document.getElementById("driverModalTitle").textContent = tr("driver_addTitle");
    document.getElementById("driverModal").style.display = "flex";
  });
  document.getElementById("btnSaveDriver").addEventListener("click", async function () {
    const backend = getBackend();
    if (!backend || !backend.SaveDriver || !backend.DeleteDriver) return;
    var original = document.getElementById("driverOriginalAgent").value;
    var agent = document.getElementById("driverAgent").value.trim();
    var name = document.getElementById("driverName").value.trim();
    var car = document.getElementById("driverCar").value.trim();
    var phone = document.getElementById("driverPhone").value.trim();
    var cities = document.getElementById("driverCities").value.trim();
    if (!agent) { log(tr("msg_enterAgent"), "error"); return; }
    try {
      if (original && original !== agent) await backend.DeleteDriver(original);
      await backend.SaveDriver({ AgentName: agent, DriverName: name, CarNumber: car, Phone: phone, CityCodes: cities });
      log(tr("msg_driverSaved"), "success");
      document.getElementById("driverModal").style.display = "none";
      loadDriversTable();
    } catch (e) {
      log("Ошибка сохранения: " + (e && e.message ? e.message : String(e)), "error");
    }
  });
  document.getElementById("btnCancelDriver").addEventListener("click", function () {
    document.getElementById("driverModal").style.display = "none";
  });
  document.getElementById("driverModal").addEventListener("click", function (e) {
    if (e.target === document.getElementById("driverModal")) document.getElementById("driverModal").style.display = "none";
  });

  async function loadItemsTable() {
    const backend = getBackend();
    if (!backend || typeof backend.GetItems !== "function") return;
    const tbody = document.getElementById("itemsTbody");
    tbody.innerHTML = "";
    try {
      const list = await backend.GetItems();
      if (!list || !list.length) {
        tbody.innerHTML = "<tr><td colspan=\"3\">" + tr("noData_items") + "</td></tr>";
        return;
      }
      list.forEach(function (it) {
        const code = (it.item_code ?? it.ItemCode ?? "").replace(/"/g, "&quot;");
        const cat = (it.category ?? it.Category ?? "").replace(/"/g, "&quot;");
        const row = document.createElement("tr");
        row.innerHTML =
          "<td>" + (it.item_code ?? it.ItemCode ?? "") + "</td><td>" + (it.category ?? it.Category ?? "") + "</td>" +
          "<td class=\"btn-cell\"><button type=\"button\" class=\"btn-secondary btn-edit-item\" data-code=\"" + code + "\" data-category=\"" + cat + "\">" + tr("btn_edit") + "</button> " +
          "<button type=\"button\" class=\"btn-secondary btn-delete-item\" data-code=\"" + code + "\">" + tr("btn_delete") + "</button></td>";
        tbody.appendChild(row);
      });
      tbody.querySelectorAll(".btn-edit-item").forEach(function (btn) {
        btn.addEventListener("click", function () {
          document.getElementById("itemOriginalCode").value = btn.getAttribute("data-code") || "";
          document.getElementById("itemCode").value = btn.getAttribute("data-code") || "";
          document.getElementById("itemCategory").value = btn.getAttribute("data-category") || "";
          document.getElementById("itemModalTitle").textContent = tr("item_editTitle");
          document.getElementById("itemModal").style.display = "flex";
        });
      });
      tbody.querySelectorAll(".btn-delete-item").forEach(function (btn) {
        btn.addEventListener("click", function () {
          var code = btn.getAttribute("data-code") || "";
          if (!confirm(trParam("msg_confirmDeleteItem", { code: code }))) return;
          var b = getBackend();
          if (!b || typeof b.DeleteItem !== "function") {
            log(tr("msg_backendUnavailable"), "error");
            return;
          }
          b.DeleteItem(code).then(function () {
            log(tr("msg_itemDeleted"), "success");
            loadItemsTable();
          }).catch(function (e) {
            log("Ошибка удаления: " + (e && e.message ? e.message : String(e)), "error");
          });
        });
      });
    } catch (e) {
      log("Ошибка загрузки товаров: " + (e && e.message ? e.message : String(e)), "error");
    }
  }

  document.getElementById("btnRefreshItems").addEventListener("click", function () {
    loadItemsTable();
    log(tr("msg_listRefreshed"));
  });
  document.getElementById("btnAddItem").addEventListener("click", function () {
    document.getElementById("itemOriginalCode").value = "";
    document.getElementById("itemCode").value = "";
    document.getElementById("itemCategory").value = "";
    document.getElementById("itemModalTitle").textContent = tr("item_addTitle");
    document.getElementById("itemModal").style.display = "flex";
  });
  document.getElementById("btnSaveItem").addEventListener("click", async function () {
    const backend = getBackend();
    if (!backend || !backend.SaveItem || !backend.DeleteItem) return;
    var original = document.getElementById("itemOriginalCode").value;
    var code = document.getElementById("itemCode").value.trim();
    var category = document.getElementById("itemCategory").value.trim();
    if (!code) { log(tr("msg_enterCode"), "error"); return; }
    try {
      if (original && original !== code) await backend.DeleteItem(original);
      await backend.SaveItem({ ItemCode: code, Category: category });
      log(tr("msg_itemSaved"), "success");
      document.getElementById("itemModal").style.display = "none";
      loadItemsTable();
    } catch (e) {
      log("Ошибка сохранения: " + (e && e.message ? e.message : String(e)), "error");
    }
  });
  document.getElementById("btnCancelItem").addEventListener("click", function () {
    document.getElementById("itemModal").style.display = "none";
  });
  document.getElementById("itemModal").addEventListener("click", function (e) {
    if (e.target === document.getElementById("itemModal")) document.getElementById("itemModal").style.display = "none";
  });

  document.getElementById("btnClearLog").addEventListener("click", function () {
    logEl.innerHTML = "";
    if (generateStatusEl) generateStatusEl.textContent = "";
  });

  if (btnOpenReportFolder) {
    btnOpenReportFolder.addEventListener("click", async function () {
      if (!lastSavedReportPath) return;
      var b = getBackend();
      if (!b || typeof b.OpenFileLocation !== "function") {
        log(tr("msg_backendUnavailable"), "error");
        return;
      }
      try {
        await b.OpenFileLocation(lastSavedReportPath);
        log(tr("msg_openFolderSuccess"), "success");
      } catch (e) {
        log("Ошибка: " + (e && e.message ? e.message : String(e)), "error");
      }
    });
  }

  function closeModalOnEscape(e) {
    if (e.key !== "Escape") return;
    var m = document.getElementById("unresolvedModal");
    if (m && m.style.display === "flex") { m.style.display = "none"; return; }
    m = document.getElementById("cityModal");
    if (m && m.style.display === "flex") { m.style.display = "none"; return; }
    m = document.getElementById("driverModal");
    if (m && m.style.display === "flex") { m.style.display = "none"; return; }
    m = document.getElementById("itemModal");
    if (m && m.style.display === "flex") { m.style.display = "none"; }
  }
  document.addEventListener("keydown", closeModalOnEscape);

  var selectedJobId = 0;

  async function refreshOutboxAndResults() {
    const backend = getBackend();
    if (!backend || typeof backend.GetOutboxJobs !== "function") return;
    try {
      var jobs = await backend.GetOutboxJobs(100);
      var tbody = document.getElementById("outboxTbody");
      if (!tbody) return;
      tbody.innerHTML = "";
      if (!jobs || !jobs.length) {
        tbody.innerHTML = "<tr><td colspan=\"4\">-</td></tr>";
      } else {
        jobs.forEach(function (j) {
          var id = j.id ?? j.ID ?? 0;
          var report = j.reportPath ?? j.ReportPath ?? "";
          var status = j.status ?? j.Status ?? "";
          var err = j.error ?? j.Error ?? "";
          var row = document.createElement("tr");
          row.innerHTML = "<td>" + id + "</td><td>" + report + "</td><td>" + status + "</td><td>" + err + "</td>";
          row.style.cursor = "pointer";
          row.addEventListener("click", function () {
            selectedJobId = id;
            loadApprovalResults(id);
          });
          tbody.appendChild(row);
        });
      }
      if (!selectedJobId && jobs && jobs.length) {
        selectedJobId = jobs[0].id ?? jobs[0].ID ?? 0;
      }
      if (selectedJobId) await loadApprovalResults(selectedJobId);
    } catch (e) {
      log((e && e.message ? e.message : String(e)), "error");
    }
  }

  async function loadApprovalResults(jobId) {
    const backend = getBackend();
    if (!backend || typeof backend.GetApprovalResults !== "function" || !jobId) return;
    try {
      var rows = await backend.GetApprovalResults(jobId);
      var approved = document.getElementById("approvedTbody");
      var rejected = document.getElementById("rejectedTbody");
      if (!approved || !rejected) return;
      approved.innerHTML = "";
      rejected.innerHTML = "";
      (rows || []).forEach(function (r) {
        var status = (r.status ?? r.Status ?? "").toLowerCase();
        var client = r.clientName ?? r.ClientName ?? "";
        var hp = r.clientHP ?? r.ClientHP ?? "";
        var inv = r.invoiceNum ?? r.InvoiceNum ?? "";
        var appr = r.approvalNum ?? r.ApprovalNum ?? "";
        var reason = r.rejectReason ?? r.RejectReason ?? "";
        var row = document.createElement("tr");
        if (status === "rejected") {
          row.innerHTML = "<td>" + client + "</td><td>" + hp + "</td><td>" + inv + "</td><td>" + reason + "</td>";
          rejected.appendChild(row);
        } else {
          row.innerHTML = "<td>" + client + "</td><td>" + hp + "</td><td>" + inv + "</td><td>" + appr + "</td>";
          approved.appendChild(row);
        }
      });
    } catch (e) {
      log((e && e.message ? e.message : String(e)), "error");
    }
  }

  var btnSaveAutomation = document.getElementById("btnSaveAutomation");
  if (btnSaveAutomation) {
    btnSaveAutomation.addEventListener("click", async function () {
      const backend = getBackend();
      if (!backend || typeof backend.SaveSettings !== "function") return;
      try {
        var raw = rawPath.value.trim();
        var out = outputDir.value.trim();
        var tpl = templatePath.value.trim();
        await backend.SaveSettings(raw, out, tpl);
        if (typeof backend.GetSettings === "function" && typeof backend.SaveAutomationSettings === "function") {
          var s = await backend.GetSettings();
          s.smtpHost = (document.getElementById("smtpHost").value || "").trim();
          s.smtpPort = parseInt((document.getElementById("smtpPort").value || "587").trim(), 10) || 587;
          s.smtpUser = (document.getElementById("smtpUser").value || "").trim();
          s.smtpPassword = (document.getElementById("smtpPassword").value || "").trim();
          s.imapHost = (document.getElementById("imapHost").value || "").trim();
          s.imapPort = parseInt((document.getElementById("imapPort").value || "993").trim(), 10) || 993;
          s.imapUser = (document.getElementById("imapUser").value || "").trim();
          s.imapPassword = (document.getElementById("imapPassword").value || "").trim();
          s.autoSend = !!(document.getElementById("autoSendEnabled").checked);
          s.watchEnabled = !!(document.getElementById("watchEnabled").checked);
          s.watchFolder = (document.getElementById("watchFolder").value || "").trim();
          s.inputFolder = s.inputFolder ?? s.InputFolder ?? raw;
          s.outputFolder = s.outputFolder ?? s.OutputFolder ?? out;
          s.templatePath = s.templatePath ?? s.TemplatePath ?? tpl;
          await backend.SaveAutomationSettings(s);
        }
        log(tr("automation_saved"), "success");
      } catch (e) {
        log((e && e.message ? e.message : String(e)), "error");
      }
    });
  }

  var btnWatchFolder = document.getElementById("btnWatchFolder");
  if (btnWatchFolder) {
    btnWatchFolder.addEventListener("click", async function () {
      const backend = getBackend();
      if (!backend || typeof backend.SelectOutputDir !== "function") return;
      try {
        var path = await backend.SelectOutputDir();
        if (path) document.getElementById("watchFolder").value = path;
      } catch (e) {}
    });
  }

  var btnScanWatchNow = document.getElementById("btnScanWatchNow");
  if (btnScanWatchNow) {
    btnScanWatchNow.addEventListener("click", async function () {
      const backend = getBackend();
      if (!backend || typeof backend.ProcessWatchFolderNow !== "function") return;
      try {
        await backend.ProcessWatchFolderNow();
        await refreshOutboxAndResults();
      } catch (e) {
        log((e && e.message ? e.message : String(e)), "error");
      }
    });
  }

  async function doResetSentRows() {
    log("Сброс счетчика: запуск...", "success");
    const backend = getBackend();
    if (!backend || typeof backend.ResetSentRowsCounter !== "function") {
      log("Сброс счетчика недоступен: метод backend не найден.", "error");
      return;
    }
    try {
      await backend.ResetSentRowsCounter();
      log(tr("automation_resetSentRows") + ": OK", "success");
    } catch (e) {
      log((e && e.message ? e.message : String(e)), "error");
    }
  }

  var btnResetSentRows = document.getElementById("btnResetSentRows");
  if (btnResetSentRows) btnResetSentRows.addEventListener("click", doResetSentRows);
  var btnQuickResetSentRows = document.getElementById("btnQuickResetSentRows");
  if (btnQuickResetSentRows) btnQuickResetSentRows.addEventListener("click", doResetSentRows);

  var btnRefreshOutbox = document.getElementById("btnRefreshOutbox");
  if (btnRefreshOutbox) {
    btnRefreshOutbox.addEventListener("click", refreshOutboxAndResults);
  }

  var btnPollRepliesNow = document.getElementById("btnPollRepliesNow");

  async function doSendNow() {
    log("Отправка: запуск...", "success");
    const backend = getBackend();
    if (!backend || typeof backend.SendReportNow !== "function") {
      log("Отправка недоступна: метод backend не найден.", "error");
      return;
    }
    try {
      var jobs = await backend.GetOutboxJobs(100);
      var selected = (jobs || []).find(function (j) { return (j.id ?? j.ID ?? 0) === selectedJobId; }) || (jobs && jobs.length ? jobs[0] : null);
      var reportPath = selected ? (selected.reportPath ?? selected.ReportPath ?? "") : (lastSavedReportPath || "");
      if (!reportPath) {
        log("Нет отчёта для отправки (выберите строку outbox или сначала сгенерируйте отчёт).", "error");
        return;
      }
      await backend.SendReportNow(reportPath);
      log("Отправка выполнена: " + reportPath, "success");
      await refreshOutboxAndResults();
    } catch (e) {
      log((e && e.message ? e.message : String(e)), "error");
    }
  }

  var btnSendSelectedNow = document.getElementById("btnSendSelectedNow");
  if (btnSendSelectedNow) btnSendSelectedNow.addEventListener("click", doSendNow);
  var btnQuickSendNow = document.getElementById("btnQuickSendNow");
  if (btnQuickSendNow) btnQuickSendNow.addEventListener("click", doSendNow);

  var btnApplyReplyText = document.getElementById("btnApplyReplyText");
  if (btnApplyReplyText) {
    btnApplyReplyText.addEventListener("click", async function () {
      const backend = getBackend();
      if (!backend || typeof backend.ApplyReplyForJob !== "function" || !selectedJobId) return;
      try {
        var text = (document.getElementById("replyTextInput").value || "").trim();
        await backend.ApplyReplyForJob(selectedJobId, text, "");
        await refreshOutboxAndResults();
      } catch (e) {
        log((e && e.message ? e.message : String(e)), "error");
      }
    });
  }

  async function doPollReplies() {
    log("Проверка ответов: запуск...", "success");
    const backend = getBackend();
    if (!backend) {
      log("Проверка ответов недоступна: backend не найден.", "error");
      return;
    }
    try {
      var count = 0;
      if (typeof backend.PollRepliesNowWithCount === "function") {
        count = await backend.PollRepliesNowWithCount();
      } else if (typeof backend.PollRepliesNow === "function") {
        await backend.PollRepliesNow();
      } else {
        log("Проверка ответов недоступна: метод backend не найден.", "error");
        return;
      }
      await refreshOutboxAndResults();
      log("Проверка ответов завершена. Обработано: " + String(count || 0), "success");
    } catch (e) {
      log((e && e.message ? e.message : String(e)), "error");
    }
  }

  var btnQuickPollReplies = document.getElementById("btnQuickPollReplies");
  if (btnQuickPollReplies) btnQuickPollReplies.addEventListener("click", doPollReplies);
  if (btnPollRepliesNow) btnPollRepliesNow.addEventListener("click", doPollReplies);

  window.onLangChange = function () {
    loadCitiesTable();
    loadDriversTable();
    loadItemsTable();
  };

  loadInitialSettings();
  loadCitiesTable();
  refreshOutboxAndResults();
  log(tr("msg_ready"));
})();
