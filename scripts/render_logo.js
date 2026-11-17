const fs = require('fs');
const path = require('path');
const { Resvg } = require('@resvg/resvg-js');
const { buildLogoSvg } = require('./generate_branding');

async function main() {
  const imgDir = path.join(__dirname, '..', 'docs', 'images');
  if (!fs.existsSync(imgDir)) {
    fs.mkdirSync(imgDir, { recursive: true });
  }

  const svg = buildLogoSvg();
  const svgPath = path.join(imgDir, 'logo.svg');
  const pngPath = path.join(imgDir, 'logo.png');

  fs.writeFileSync(svgPath, svg, 'utf8');
  console.log(`Saved SVG: ${svgPath}`);

  const resvg = new Resvg(svg, {
    fitTo: { mode: 'width', value: 1024 }
  });
  const pngData = resvg.render().asPng();
  fs.writeFileSync(pngPath, pngData);
  console.log(`Rendered PNG (1024x1024): ${pngPath}`);
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
