from pathlib import Path
import subprocess
import sys
import tempfile
from PIL import Image
source, target, width, height = sys.argv[1:]
target=Path(target)
target.parent.mkdir(parents=True, exist_ok=True)
with tempfile.TemporaryDirectory(prefix='turkiye-avif-') as temp:
 resized=Path(temp)/'resized.png'
 encoded=Path(temp)/'encoded.avif'
 subprocess.run(['magick',source,'-resize',f'{width}x{height}!','PNG:'+str(resized)],check=True,capture_output=True)
 subprocess.run(['avifenc','-q','85','-s','6','-j','2','-y','444',str(resized),str(encoded)],check=True,capture_output=True)
 with Image.open(encoded) as im:
  im.load()
  assert im.format=='AVIF' and im.size==(int(width),int(height)), (im.format,im.size)
 temporary=target.with_name('.'+target.name+'.tmp')
 temporary.write_bytes(encoded.read_bytes())
 temporary.replace(target)
 print(target, 'AVIF', width, height)
