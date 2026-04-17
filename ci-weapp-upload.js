const ci = require("miniprogram-ci");
const path = require("path");
const { appid } = require("../project.config.json");

// 从环境变量获取，避免密钥泄露
const PRIVATE_KEY = process.env.WX_UPLOAD_PRIVATE_KEY;
const VERSION = process.env.WX_UPLOAD_VERSION;

if (!PRIVATE_KEY || !VERSION) {
  console.error("[error] 缺少环境变量: WX_UPLOAD_PRIVATE_KEY 或 WX_UPLOAD_VERSION 未设置（请检查 Jenkins 凭据 ID: wx-upload-private-key）");
  process.exit(1);
}

const DESCRIPTION = `Build by Jenkins #${VERSION}`;
const APP_ID = appid;

const project = new ci.Project({
  appid: APP_ID,
  type: "miniProgram",
  projectPath: path.join(__dirname, "dist"), // build后的目录
  privateKey: PRIVATE_KEY, // 与 miniprogram-ci 一致：内容用 privateKey，文件路径用 privateKeyPath
  ignores: ["node_modules/**/*", "README.md"],
});

async function upload() {
  console.log("--- 开始上传至微信后台 ---");
  console.log("VERSION: ", VERSION);
  console.log("DESCRIPTION: ", DESCRIPTION);
  console.log("APP_ID: ", APP_ID);

  const uploadResult = await ci.upload({
    project,
    version: VERSION,
    desc: DESCRIPTION,
    setting: {
      es6: true,
      minify: true,
    },
    onProgressUpdate: console.log,
  });
  console.log("--- 上传成功 ---", uploadResult);
}

upload().catch((err) => {
  console.error("--- 上传失败 ---", err);
  process.exit(1);
});
