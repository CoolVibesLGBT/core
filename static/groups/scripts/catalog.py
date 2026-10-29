from pathlib import Path
from PIL import Image
import json
ROOT=Path(__file__).resolve().parent.parent
KINDS={'avatar':(512,512),'cover':(1500,500),'hero':(1920,1280)}
briefs={b['code']:b for file in ('briefs-a.json','briefs-b.json') for b in json.loads((ROOT/'data'/file).read_text())}
entries=[]; saved=0; errors=[]
for row in json.loads((ROOT/'data/countries.json').read_text()):
 code=row['code']; brief=briefs[code]
 entry={**row,'description':', '.join(x['name'] for x in brief['landmarks'])+'. Bölgesel bir seçkidir; eksiksiz envanter değildir.','animals':', '.join(brief['animals']),'depthMode':'approximate','status':'pending','sources':brief['sources']}
 for kind,size in KINDS.items():
  path=ROOT/'images'/f'{code}_{kind}.avif'
  if path.exists():
   try:
    with Image.open(path) as im:
     im.load()
     assert im.format=='AVIF' and im.size==size,(im.format,im.size)
    entry[kind]=str(path.relative_to(ROOT)); saved+=1
   except Exception as exc:errors.append(f'{path.name}: {exc}')
 entry['depth']={kind:f'depth/{code}_{kind}.png' for kind in KINDS if (ROOT/'depth'/f'{code}_{kind}.png').is_file()}
 entry['visualReview']='recorded' if (ROOT/'production'/f'{code}_review.json').is_file() else 'pending'
 count=sum(kind in entry for kind in KINDS)
 entry['status']='complete' if count==3 else 'partial' if count else 'pending'
 entries.append(entry)
catalog={'scope':'Countries and regions: ISO 3166-1, additional special-status places and UK constituent countries; inclusion is not a sovereignty claim','totalCountries':len(entries),'requiredAssets':len(entries)*len(KINDS),'savedAssets':saved,'completeCountries':sum(x['status']=='complete' for x in entries),'countries':entries,'errors':errors}
(ROOT/'data/catalog.json').write_text(json.dumps(catalog,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({k:catalog[k] for k in ('totalCountries','requiredAssets','savedAssets','completeCountries','errors')},ensure_ascii=False))
