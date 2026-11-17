const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');

function main() {
  const rootDir = path.join(__dirname, '..');
  const framesPattern = path.join(rootDir, 'docs', 'images', 'frames', 'frame_%04d.png');
  const gifOut = path.join(rootDir, 'docs', 'images', 'queryproof_demo.gif');

  console.log('Generating optimized GIF with ffmpeg...');
  const cmd = `ffmpeg -y -framerate 6 -i "${framesPattern}" -vf "scale=1200:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=128:reserve_transparent=0[p];[s1][p]paletteuse=dither=bayer:bayer_scale=5" "${gifOut}"`;

  execSync(cmd, { stdio: 'inherit' });
  const stat = fs.statSync(gifOut);
  console.log(`✓ Successfully generated ${gifOut} (${(stat.size / (1024 * 1024)).toFixed(2)} MB)`);

  // Clean up frames directory to keep git repository lean
  const framesDir = path.join(rootDir, 'docs', 'images', 'frames');
  if (fs.existsSync(framesDir)) {
    fs.rmSync(framesDir, { recursive: true, force: true });
    console.log('✓ Cleaned up temporary frames directory');
  }
}

main();
