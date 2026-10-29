import sys
from pathlib import Path
from PIL import Image, ImageDraw
codes=sys.argv[1:]
canvas=Image.new('RGB',(1800,430*len(codes)),'#eee')
draw=ImageDraw.Draw(canvas)
for i,c in enumerate(codes):
 for j,k in enumerate(['hero','cover','avatar']):
  path=Path(f'images/{c}_{k}.avif')
  if not path.exists():continue
  im=Image.open(path);im.thumbnail((590,400))
  canvas.paste(im,(j*600,i*430+25))
  draw.text((j*600+8,i*430+6),f'{c}_{k}',fill='black')
path=Path('production/expansion-reviews')/('-'.join(codes)+'.jpg')
canvas.save(path)
print(path)
