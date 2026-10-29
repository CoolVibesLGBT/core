"""Check external and embedded asset availability match, including partial countries."""
from pathlib import Path
import json, re, shutil, subprocess, sys, tempfile
root=Path(__file__).resolve().parent.parent
with tempfile.TemporaryDirectory(prefix='atlas-build-qa-') as tmp:
    fixture=Path(tmp)
    for directory in ('scripts','data','images'): (fixture/directory).mkdir()
    shutil.copy(root/'scripts/build-page.py',fixture/'scripts/build-page.py')
    shutil.copytree(root/'scripts/runtime',fixture/'scripts/runtime')
    shutil.copy(root/'page.template.html',fixture/'page.template.html')
    (fixture/'data/countries.json').write_text(json.dumps([{'code':'tr','name':'Türkiye'},{'code':'jp','name':'Japan'}]))
    # Deliberately only one asset; catalog also references absent assets and depth.
    (fixture/'images/tr_cover.avif').write_bytes(b'fixture avif')
    (fixture/'data/catalog.json').write_text(json.dumps({'countries':[{'code':'tr','avatar':'images/absent.avif','hero':'images/missing.avif','depth':{'cover':'depth/absent.png'}}]}))
    subprocess.run([sys.executable,str(fixture/'scripts/build-page.py')],check=True,capture_output=True)
    def assets(name):
        text=(fixture/name).read_text()
        return json.loads(re.search(r'const ASSETS = (.*?);\nconst scene=',text,re.S).group(1))
    embedded,external=assets('index.html'),assets('atlas.html')
    assert set(embedded)==set(external)=={'tr'}
    assert set(embedded['tr'])==set(external['tr'])=={'cover'}
    assert external['tr']['cover']=='images/tr_cover.avif'
    assert embedded['tr']['cover'].startswith('data:image/avif;base64,')
    embedded_html=(fixture/'index.html').read_text();external_html=(fixture/'atlas.html').read_text()
    assert '/* ATLAS_RUNTIME */' not in embedded_html and '/* EMBEDDED_ASSETS */' not in external_html
    assert embedded_html.split('const scene=',1)[1]==external_html.split('const scene=',1)[1]
    for component in ('class GPUFlow','class CanvasVideoRecorder','class AdaptiveQuality','function beginVisualTransition'):
        assert component in embedded_html
print('PASS: partial-country external/embedded availability match; missing assets excluded; identical inline runtime and all engine modules bundled.')
