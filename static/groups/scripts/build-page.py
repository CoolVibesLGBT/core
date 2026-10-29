"""Embed the atlas and its completed AVIF assets for a standalone file:// preview."""
from pathlib import Path
import base64, json
root=Path(__file__).resolve().parent.parent

def embedded(path):
    p=Path(path)
    if not p.is_absolute(): p=root/p
    if not p.is_file(): return None
    mime={'avif':'image/avif','png':'image/png','jpg':'image/jpeg','webp':'image/webp'}.get(p.suffix[1:],'application/octet-stream')
    return 'data:'+mime+';base64,'+base64.b64encode(p.read_bytes()).decode()

country_file=root/'data/countries.json'
raw=json.loads(country_file.read_text()) if country_file.exists() else [{'code':'tr','name':'Türkiye'}]
countries=raw.get('countries',[]) if isinstance(raw,dict) else raw
countries=[{**c,'code':str(c.get('code',c.get('alpha2',''))).lower(),'name':c.get('name_tr',c.get('name',''))} for c in countries]
catalog_file=root/'data/catalog.json'
catalog=json.loads(catalog_file.read_text()) if catalog_file.exists() else []
if isinstance(catalog,dict):
    catalog=catalog.get('countries',catalog.get('items',catalog))
if isinstance(catalog,dict): catalog=[dict(v,code=k) for k,v in catalog.items() if isinstance(v,dict)]
metadata={str(c.get('code','')).lower():c for c in catalog}
assets={}
for c in countries:
    code=c['code']; meta=metadata.get(code,{})
    item={k:meta[k] for k in ('description','animals','depthMode') if k in meta}
    for kind in ('cover','avatar','hero'):
        path=meta.get(kind) or meta.get('assets',{}).get(kind) or f'images/{code}_{kind}.avif'
        if isinstance(path,dict):path=path.get('path','')
        value=embedded(path) if path else None
        if value:item[kind]=value
    depth=meta.get('depth',{})
    if isinstance(depth,dict):
        values={k:embedded(v) for k,v in depth.items() if k in ('cover','avatar','hero') and isinstance(v,str)}
        if any(values.values()):item['depth']={k:v for k,v in values.items() if v}
    if any(k in item for k in ('cover','avatar','hero')):assets[code]=item
payload='const COUNTRIES = '+json.dumps(countries,ensure_ascii=False)+';\nconst ASSETS = '+json.dumps(assets,ensure_ascii=False)+';'
runtime_files=['state.js','flow.js','renderer.js','motion.js','features.js','viewer.js','controls.js','boot.js']
runtime='\n'.join((root/'scripts/runtime'/name).read_text() for name in runtime_files)
template=(root/'page.template.html').read_text().replace('/* ATLAS_RUNTIME */',runtime)
(root/'index.html').write_text(template.replace('/* EMBEDDED_ASSETS */',payload))
# Smaller external-asset edition for the full catalog. Serve this file over local HTTP.
external={}
for code,item in assets.items():
    out={k:v for k,v in item.items() if k not in ('cover','avatar','hero','depth')}
    meta=metadata.get(code,{})
    for kind in ('cover','avatar','hero'):
        if kind not in item: continue
        path=meta.get(kind) or meta.get('assets',{}).get(kind) or f'images/{code}_{kind}.avif'
        out[kind]=path.get('path','') if isinstance(path,dict) else path
    if item.get('depth'):out['depth']={k:meta['depth'][k] for k in item['depth']}
    external[code]=out
external_payload='const COUNTRIES = '+json.dumps(countries,ensure_ascii=False)+';\nconst ASSETS = '+json.dumps(external,ensure_ascii=False)+';'
(root/'atlas.html').write_text(template.replace('/* EMBEDDED_ASSETS */',external_payload))
print(f'index.html: {len(assets)} countries with assets / {len(countries)} countries; embedded AVIF assets')
