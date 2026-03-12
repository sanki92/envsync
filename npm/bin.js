const { execFileSync } = require("child_process");
const path = require("path");
const fs = require("fs");
const os = require("os");
const https = require("https");

const VERSION = require("./package.json").version;
const REPO = "sanki92/envsync";

function getPlatform() {
  const platform = os.platform();
  const arch = os.arch();

  const platformMap = {
    darwin: "darwin",
    linux: "linux",
    win32: "windows",
  };

  const archMap = {
    x64: "amd64",
    arm64: "arm64",
  };

  const goos = platformMap[platform];
  const goarch = archMap[arch];

  if (!goos || !goarch) {
    console.error(`Unsupported platform: ${platform}/${arch}`);
    process.exit(1);
  }

  return { goos, goarch };
}

function getBinaryName() {
  const { goos, goarch } = getPlatform();
  const ext = goos === "windows" ? ".exe" : "";
  return `envsync_${goos}_${goarch}${ext}`;
}

function getBinaryNames() {
  const primary = getBinaryName();
  if (os.platform() !== "win32") {
    return [primary];
  }

  // v0.1.0 Windows release assets were uploaded with a double .exe suffix.
  return [primary, `${primary}.exe`];
}

function getBinaryPath() {
  const cacheDir = path.join(os.homedir(), ".envsync", "bin");
  if (!fs.existsSync(cacheDir)) {
    fs.mkdirSync(cacheDir, { recursive: true });
  }
  const ext = os.platform() === "win32" ? ".exe" : "";
  return path.join(cacheDir, `envsync${ext}`);
}

function download(url) {
  return new Promise((resolve, reject) => {
    https
      .get(url, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          return download(res.headers.location).then(resolve).catch(reject);
        }
        if (res.statusCode !== 200) {
          return reject(new Error(`HTTP ${res.statusCode} for ${url}`));
        }
        const chunks = [];
        res.on("data", (chunk) => chunks.push(chunk));
        res.on("end", () => resolve(Buffer.concat(chunks)));
        res.on("error", reject);
      })
      .on("error", reject);
  });
}

async function ensureBinary() {
  const binPath = getBinaryPath();

  if (fs.existsSync(binPath)) {
    return binPath;
  }

  console.error(`Downloading envsync v${VERSION}...`);

  const errors = [];
  for (const binaryName of getBinaryNames()) {
    const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${binaryName}`;

    try {
      const data = await download(url);
      fs.writeFileSync(binPath, data, { mode: 0o755 });
      console.error("Done.");
      return binPath;
    } catch (err) {
      errors.push(`${url} (${err.message})`);
    }
  }

  console.error("Failed to download binary.");
  for (const error of errors) {
    console.error(`  ${error}`);
  }
  console.error("");
  console.error("You can build from source:");
  console.error("  git clone https://github.com/sanki92/envsync.git");
  console.error("  cd envsync && go build -o envsync .");
  process.exit(1);
}

async function main() {
  const binPath = await ensureBinary();
  const args = process.argv.slice(2);

  try {
    execFileSync(binPath, args, { stdio: "inherit" });
  } catch (err) {
    process.exit(err.status || 1);
  }
}

main();
