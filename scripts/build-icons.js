const fs = require("fs");
const path = require("path");
const sharp = require("sharp");
const png2icons = require("png2icons");

const rootDir = path.resolve(__dirname, "..");
const svgPath = path.join(rootDir, "web", "public", "robot.svg");
const outDir = path.join(rootDir, "build", "icons");
const pngPath = path.join(outDir, "icon.png");
const icnsPath = path.join(outDir, "icon.icns");
const icoPath = path.join(outDir, "icon.ico");

async function main() {
  if (!fs.existsSync(svgPath)) {
    throw new Error(`Missing SVG icon at ${svgPath}`);
  }

  fs.mkdirSync(outDir, { recursive: true });

  const pngBuffer = await sharp(svgPath)
    .resize(1024, 1024, { fit: "contain" })
    .png()
    .toBuffer();

  fs.writeFileSync(pngPath, pngBuffer);

  const icns = png2icons.createICNS(pngBuffer, png2icons.BICUBIC, false);
  if (!icns) {
    throw new Error("Failed to generate ICNS icon");
  }
  fs.writeFileSync(icnsPath, icns);

  const ico = png2icons.createICO(pngBuffer, png2icons.BICUBIC, false);
  if (!ico) {
    throw new Error("Failed to generate ICO icon");
  }
  fs.writeFileSync(icoPath, ico);

  console.log("Icons generated:");
  console.log(pngPath);
  console.log(icnsPath);
  console.log(icoPath);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
