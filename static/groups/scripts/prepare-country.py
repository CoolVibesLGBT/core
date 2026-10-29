from pathlib import Path
import json,sys
ROOT=Path(__file__).resolve().parent.parent
briefs=json.loads((ROOT/'data/briefs.json').read_text())
base=(ROOT/'production/style.txt').read_text().strip()
palettes=json.loads((ROOT/'production/palettes.json').read_text())
code=sys.argv[1].lower();spec=next(b for b in briefs if b['code']==code)
palette=' '.join(palettes[next(i for i,b in enumerate(briefs) if b['code']==code)%len(palettes)])
result=[]
for kind,size in [('hero',[1920,1280]),('cover',[1500,500]),('avatar',[512,512])]:
 if kind=='avatar':
  accents=(', a few '+', '.join(spec['plants'][:2])+' accents') if spec['plants'] else ', with bare rock and ice accents only; no trees, flowers or invented vegetation'
  scene='Focus ONLY on '+spec['avatarFocus']+' with one natural '+spec['animals'][0]+accents+' and an atmospheric hint of the region landscape. Large iconic readable shapes, generous central circular crop safe margin. Square1:1 final512x512.'
 else:
  scene='Show these authentic landmarks in distinct regional clusters: '+'; '.join(l['name']+' ('+l['region']+')' for l in spec['landmarks'])+'. Natural animals in appropriate separate habitats: '+', '.join(spec['animals'])+'. '
  scene+=('Local plants: '+', '.join(spec['plants'])+'. ') if spec['plants'] else 'Bare rock, snow and ice only; no invented trees, flowers or vegetation. '
  if spec['foodOrCraft']:scene+='Small optional authentic craft/food detail: '+', '.join(spec['foodOrCraft'])+'. '
  scene+= 'Hero3:2 final1920x1280. Deep layered diorama: prominent near animal lower-right and near plant framing bottom-left, landmark groups spread middle ground, geographic landscape and sky behind.' if kind=='hero' else 'Cover panorama3:1 final1500x500. Clear broad region clusters from left to right, animal foreground near lower-right and plants lower-left; ample upper breathing room, mountains and sky far behind.'
 prompt=base+' Country: '+spec['name']+'. Palette from supplied collection: '+palette+', supported by muted natural stone, wood and vegetation colors with purple shadows. '+scene
 path=ROOT/'production'/f'{code}_{kind}_prompt.txt';path.write_text(prompt+'\n')
 result.append({'code':code,'kind':kind,'size':size,'prompt':prompt,'output':f'images/{code}_{kind}.avif','exists':(ROOT/'images'/f'{code}_{kind}.avif').exists()})
print(json.dumps(result,ensure_ascii=False))
