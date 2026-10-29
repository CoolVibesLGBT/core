import json,sys,pathlib,subprocess
briefs={x['code']:x for x in json.load(open('data/briefs-expansion.json'))}
for row in json.loads(sys.argv[1]):
 c,k,*ellipses=row
 regions=[{'ellipse':e,'value':245 if i==0 else 225} for i,e in enumerate(ellipses)]
 regions.append({'polygon':[[0,.72],[.16,.79],[.36,1],[0,1]],'value':250})
 p=pathlib.Path(f'production/{c}_{k}_source.json');source=json.loads(p.read_text());b=briefs[c]
 notes='Visually inspected detailed painterly atlas style, complete foreground wildlife silhouettes and principal landmark framing. Regional selection: '+', '.join(l['name'] for l in b['landmarks'])+'. No lettering or excluded religious architecture observed at review scale. Wildlife stylized. Approximate depth masks, not measured geometry.'
 subprocess.run(['python3','scripts/save-reviewed-asset-b.py',c,k,source['path'],notes,json.dumps(regions)],check=True)
 p.write_text(json.dumps(source,indent=2))
