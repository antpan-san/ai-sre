#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";

const rawBaseURL = process.env.APP_BASE_URL || "http://127.0.0.1:9080";
const appBaseURL = rawBaseURL.replace(/\/ft-api\/?$/, "").replace(/\/$/, "");
const apiBaseURL = `${appBaseURL}/ft-api`;
const username = process.env.APP_USERNAME || "admin";
const password = process.env.APP_PASSWORD || "password";
const headless = process.env.APP_SMOKE_HEADLESS !== "0";
const browserExecutable = [
  process.env.APP_SMOKE_BROWSER_PATH,
  "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
  "/Applications/Chromium.app/Contents/MacOS/Chromium",
  "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
].find((candidate) => candidate && fs.existsSync(candidate));

async function loadPlaywright() {
  const candidates = [
    process.env.PLAYWRIGHT_MODULE_PATH,
    path.join(
      process.env.HOME || "",
      ".cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/.pnpm/playwright@1.60.0/node_modules/playwright/index.js"
    ),
  ].filter(Boolean);

  try {
    return await import("playwright");
  } catch {
    for (const candidate of candidates) {
      if (candidate && fs.existsSync(candidate)) {
        return import(pathToFileURL(candidate).href);
      }
    }
    throw new Error(
      "无法加载 playwright。请安装依赖，或设置 PLAYWRIGHT_MODULE_PATH 指向可用的 playwright/index.js"
    );
  }
}

function resolveChromium(module) {
  return module?.chromium || module?.default?.chromium || null;
}

function solveCaptcha(challenge) {
  const match = String(challenge || "").match(/(\d+)\s*\+\s*(\d+)/);
  if (!match) return "";
  return String(Number(match[1]) + Number(match[2]));
}

async function apiJson(url, options = {}) {
  const resp = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
  });
  const text = await resp.text();
  let json = {};
  try {
    json = text ? JSON.parse(text) : {};
  } catch {
    throw new Error(`接口 ${url} 返回非 JSON：${text.slice(0, 180)}`);
  }
  return { resp, json };
}

function assert(ok, message) {
  if (!ok) throw new Error(message);
}

function pickExecutionId(payload) {
  const maybeLists = [
    payload?.data?.records,
    payload?.data?.list,
    payload?.data?.items,
    payload?.records,
    payload?.list,
    payload?.items,
  ];
  for (const list of maybeLists) {
    if (Array.isArray(list) && list.length && list[0]?.id) return String(list[0].id);
  }
  return "";
}

async function login() {
  const { json: authOptions } = await apiJson(`${apiBaseURL}/api/auth/public-options`);
  const opts = authOptions?.data || authOptions || {};

  let captcha_id = "";
  let captcha_answer = "";
  if (opts.login_captcha_required) {
    const { json: captcha } = await apiJson(`${apiBaseURL}/api/auth/login-captcha`);
    const data = captcha?.data || captcha || {};
    captcha_id = String(data.captcha_id || "");
    captcha_answer = solveCaptcha(data.challenge);
  }

  const { json: loginResp } = await apiJson(`${apiBaseURL}/api/auth/login`, {
    method: "POST",
    body: JSON.stringify({ username, password, captcha_id, captcha_answer }),
  });

  assert(loginResp?.code === 200, `登录失败：${loginResp?.msg || loginResp?.message || "unknown error"}`);
  const data = loginResp?.data || {};
  assert(data.token, "登录成功但未返回 token");
  assert(data.user, "登录成功但未返回 user 信息");
  return data;
}

async function main() {
  const chromium = resolveChromium(await loadPlaywright());
  assert(chromium, "无法从 playwright 模块解析 chromium");
  const auth = await login();
  const { json: execResp } = await apiJson(`${apiBaseURL}/api/execution-records?page=1&pageSize=1`, {
    headers: { Authorization: `Bearer ${auth.token}` },
  });
  const executionId = pickExecutionId(execResp);

  const routeChecks = [
    { path: "/app/dashboard", expect: "概览" },
    { path: "/app/execution-records", expect: "执行记录" },
    ...(executionId ? [{ path: `/app/executions/${executionId}`, expect: "" }] : []),
    { path: "/app/workloads", expect: "工作负载" },
    { path: "/app/capabilities", expect: "能力中心" },
    { path: "/app/troubleshooting", expect: "问题排查" },
    { path: "/app/job/center", expect: "作业中心" },
    { path: "/app/settings", expect: "设置" },
    { path: "/app/service/deploy", expect: "基础服务" },
    { path: "/app/service/deploy/nginx", expect: "Nginx" },
    { path: "/app/service/k8s-deploy", expect: "Kubernetes" },
    { path: "/app/service/k8s-deploy/progress", expect: "" },
    { path: "/app/service/linux", expect: "Linux 主机" },
    { path: "/app/k8s-mirror", expect: "制品目录" },
    { path: "/app/init-tools", expect: "节点初始化" },
    { path: "/app/monitoring/prometheus", expect: "Prometheus" },
    { path: "/app/monitoring/node-exporter", expect: "Node Exporter" },
    { path: "/app/monitoring/jmx-exporter", expect: "JMX Exporter" },
    { path: "/app/monitoring/redis-exporter", expect: "Redis Exporter" },
    { path: "/app/monitoring/mongodb-exporter", expect: "MongoDB Exporter" },
    { path: "/app/monitoring/blackbox-exporter", expect: "Blackbox Exporter" },
    { path: "/app/advanced/backup-restore", expect: "备份" },
    { path: "/app/advanced/performance-analysis", expect: "性能" },
    { path: "/app/advanced/runtime-observe", expect: "运行时诊断" },
    { path: "/app/help/error-codes", expect: "错误码" },
  ];

  const browser = await chromium.launch({
    headless,
    ...(browserExecutable ? { executablePath: browserExecutable } : {}),
  });
  const context = await browser.newContext({ ignoreHTTPSErrors: true });
  await context.addInitScript(
    ([token, user]) => {
      localStorage.setItem("token", token);
      localStorage.setItem("userInfo", JSON.stringify(user));
    },
    [auth.token, auth.user]
  );
  const page = await context.newPage();
  const routeErrors = [];
  let currentPath = "";

  page.on("pageerror", (error) => {
    routeErrors.push(`[${currentPath}] pageerror: ${error.message}`);
  });
  page.on("console", (msg) => {
    if (msg.type() === "error") {
      routeErrors.push(`[${currentPath}] console: ${msg.text()}`);
    }
  });

  const results = [];

  for (const check of routeChecks) {
    currentPath = check.path;
    routeErrors.length = 0;
    await page.goto(`${appBaseURL}${check.path}`, { waitUntil: "domcontentloaded" });
    await page.waitForLoadState("networkidle").catch(() => {});
    await page.waitForTimeout(350);

    const currentUrl = page.url();
    assert(!currentUrl.includes("/login"), `${check.path} 被重定向到登录页`);

    const bodyText = await page.locator("body").innerText();
    assert(
      bodyText.trim().length > 40,
      `${check.path} 页面内容过少，疑似白屏${routeErrors.length ? `：\n${routeErrors.join("\n")}` : ""}`
    );
    if (check.expect) {
      assert(bodyText.includes(check.expect), `${check.path} 未找到预期文本：${check.expect}`);
    }
    assert(routeErrors.length === 0, `${check.path} 存在前端错误：\n${routeErrors.join("\n")}`);

    results.push(`OK ${check.path}`);
  }

  currentPath = "/app/workloads";
  routeErrors.length = 0;
  await page.goto(`${appBaseURL}/app/workloads`, { waitUntil: "domcontentloaded" });
  await page.waitForLoadState("networkidle").catch(() => {});
  const nginxTile = page.locator(".workload-tile").filter({ hasText: "Nginx" }).first();
  assert((await nginxTile.count()) > 0, "工作负载页未找到 Nginx 服务卡");
  await nginxTile.getByRole("button", { name: "进入" }).click();
  await page
    .waitForURL((url) => url.pathname.includes("/app/service/deploy/"), { timeout: 10000 })
    .catch(() => {});
  await page.waitForLoadState("networkidle").catch(() => {});
  await page.waitForTimeout(300);
  assert(page.url().includes("/app/service/deploy/"), "服务卡未直接进入独立服务页");
  assert(routeErrors.length === 0, `工作负载服务直达存在前端错误：\n${routeErrors.join("\n")}`);

  console.log(`smoke base: ${appBaseURL}`);
  console.log(`login user: ${username}`);
  if (!executionId) {
    console.log("note: 未找到执行记录，已跳过 /app/executions/:id 动态页检查");
  }
  for (const line of results) console.log(line);
  console.log("OK workload service card direct route");

  await browser.close();
}

main().catch((error) => {
  console.error(`smoke:app failed: ${error instanceof Error ? error.message : String(error)}`);
  process.exitCode = 1;
});
