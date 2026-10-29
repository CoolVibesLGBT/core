"""Render hand-authored approximate depth DATA; never modifies the color artwork.
Usage: python3 scripts/depth-map.py data/depth-regions/xx_hero.json
JSON: {output,width,height,stops:[[y,value],...],regions:[{polygon:[[x,y],...],value,blur}|{ellipse:[cx,cy,rx,ry],value,blur}]}.
Positions are normalized to 0..1; near=255, far=0.
"""
from pathlib import Path
import json,sys
from PIL import Image,ImageDraw,ImageFilter
spec=json.loads(Path(sys.argv[1]).read_text()); w=spec.get('width',512);h=spec.get('height',342)
stops=spec.get('stops',[[0,20],[.25,40],[.5,110],[1,225]])
im=Image.new('L',(w,h)); draw=ImageDraw.Draw(im)
for y in range(h):
 t=y/(h-1);i=next((i for i in range(len(stops)-1) if stops[i][0]<=t<=stops[i+1][0]),len(stops)-2)
 a,b=stops[i:i+2];v=round(a[1]+(b[1]-a[1])*(t-a[0])/(b[0]-a[0]));draw.line((0,y,w,y),fill=max(0,min(255,v)))
for region in spec.get('regions',[]):
 layer=Image.new('L',(w,h),0);d=ImageDraw.Draw(layer)
 if 'polygon' in region:d.polygon([(round(x*w),round(y*h)) for x,y in region['polygon']],fill=255)
 if 'ellipse' in region:
  cx,cy,rx,ry=region['ellipse'];d.ellipse(((cx-rx)*w,(cy-ry)*h,(cx+rx)*w,(cy+ry)*h),fill=255)
 layer=layer.filter(ImageFilter.GaussianBlur(region.get('blur',.006)*w))
 im.paste(region['value'],(0,0,w,h),layer)
out=Path(spec['output']);out.parent.mkdir(parents=True,exist_ok=True);im.save(out)
print(out)
