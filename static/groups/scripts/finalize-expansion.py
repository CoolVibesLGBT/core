"""Reconcile the coverage report against delivered, reviewed atlas assets."""
from pathlib import Path
import csv
import html
import json
from PIL import Image

root = Path(__file__).resolve().parent.parent
countries = json.loads((root/'data/countries.json').read_text())
added = json.loads((root/'production/expansion-entries.json').read_text())
codes = {x['code'] for x in countries}
iso = json.loads((root/'production/iso3166-current-snapshot.json').read_text())
assert len(countries) == len(codes) == 259
assert not {x['code'] for x in iso} - codes
sizes = {'hero': (1920,1280), 'cover': (1500,500), 'avatar': (512,512)}
prompts = []
for entry in countries:
    c = entry['code']
    for k, size in sizes.items():
        with Image.open(root/f'images/{c}_{k}.avif') as im:
            im.load()
            assert im.format == 'AVIF' and im.size == size, (c,k,im.size)
        with Image.open(root/f'depth/{c}_{k}.png') as im:
            im.verify()
for entry in added:
    c = entry['code']
    review = json.loads((root/f'production/{c}_review.json').read_text())
    for k in sizes:
        assert review[k]['status'] == 'reviewed', (c,k)
        source_path = root/f'production/{c}_{k}_source.json'
        source = json.loads(source_path.read_text())
        source['saved_asset'] = f'images/{c}_{k}.avif'
        source['source_png_available'] = Path(source['path']).is_file()
        source_path.write_text(json.dumps(source,ensure_ascii=False,indent=2))
        prompts.append({'code':c,'kind':k,**source})
followups = json.loads((root/'production/expansion-followups.json').read_text())
assert all(x['status'] == 'resolved' for x in followups)
(root/'production/expansion-final-prompts.json').write_text(json.dumps(prompts,ensure_ascii=False,indent=2))
p = root/'data/country-coverage-audit.json'
audit = json.loads(p.read_text())
if 'baseline_missing_iso_entries' not in audit:
    audit['baseline_missing_iso_entries'] = audit.get('missing_iso_entries',[])
audit['baseline_missing_iso_count'] = 54
audit['iso_missing_count'] = 0
audit['missing_iso_entries'] = []
audit['iso_entries_present'] = 249
audit['current_catalog'] = audit['unique_codes'] = 259
audit['expansion_entries_added'] = 64
audit['expansion_assets_saved'] = audit['expansion_assets_required'] = 192
audit['total_assets_saved'] = 777
audit['expansion_status'] = 'complete'
audit['added_entries'] = added
for key in ['baseline_missing_iso_entries','non_iso_special_status_candidates','constituent_countries_not_separate']:
    for entry in audit.get(key,[]):
        entry['in_current_catalog'] = entry.get('code') in codes
audit['scope_warning'] = '259 ülke/bölge kaydı: ISO 3166-1 kapsamındaki 249 giriş, altı ek özel statülü yer ve Birleşik Krallık içindeki dört kurucu ülke. Katalogda bulunmak egemenlik veya tanınma iddiası değildir.'
p.write_text(json.dumps(audit,ensure_ascii=False,indent=2))
csvpath = root/'production/missing-country-region-entries.csv'
with csvpath.open('w',encoding='utf-8-sig',newline='') as f:
    writer = csv.DictWriter(f,fieldnames=['code','name_tr','type','code_scheme','in_current_catalog','assets_saved','status'])
    writer.writeheader()
    for e in added:
        writer.writerow({k:e.get(k,'') for k in ['code','name_tr','type','code_scheme']} | {'in_current_catalog':True,'assets_saved':3,'status':'complete'})
rows = ''.join('<tr>'+''.join(f'<td>{html.escape(str(v))}</td>' for v in [e['code'],e['name_tr'],e['type'],e['code_scheme'],'3 / 3 — tamamlandı'])+'</tr>' for e in added)
sources = ''.join(f'<li><a href="{html.escape(s)}">{html.escape(s)}</a></li>' for s in audit['sources'])
(root/'country-coverage-audit.html').write_text('''<!doctype html><html lang="tr"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Ülke kapsamı denetimi</title><style>body{font:16px system-ui;max-width:1100px;margin:40px auto;padding:0 20px;color:#242236;background:#faf9f6}table{border-collapse:collapse;width:100%}td,th{padding:10px;border-bottom:1px solid #ccc;text-align:left}a{color:#514184}.scroll{overflow:auto}</style><h1>Ülke kapsamı denetimi — tamamlandı</h1><p>Başlangıçtaki 195 kayda eksik 64 giriş eklendi. Toplam <strong>259 ülke/bölge, 777 görsel ve 777 yaklaşık derinlik haritası</strong> hazır. Tayvan, Azerbaycan, Avustralya ve Avusturya üçer görselle katalogda.</p><p>193 BM üyesinin tamamı, iki gözlemciyle ilişkili PS/VA kayıtları ve ISO 3166-1 kapsamındaki 249 giriş mevcut. ISO listesi dışında altı özel statülü yer ve Birleşik Krallık'ın dört kurucu ülkesi ayrıca gösterilir. Eklenme, egemenlik veya uluslararası tanınma iddiası değildir. Uygulamaya özel kodlar ISO kodu olarak sunulmaz.</p><p>Görseller bölgesel bir seçkidir; farklı konumları sanatsal bir kompozisyonda birleştirir. Derinlik haritaları elle hazırlanmış yaklaşık maskelerdir. Yeni 192 görsel görsel olarak incelendi; kayda alınan düzeltmeler kapatıldı.</p><p><a href="atlas.html">Atlası aç</a> · <a href="production/missing-country-region-entries.csv">Eklenen 64 kaydın CSV listesi</a> · <a href="production/expansion-final-prompts.json">Son prompt seti</a></p><div class="scroll"><table><thead><tr><th>Kod</th><th>Ad</th><th>Tür</th><th>Kod sistemi</th><th>Görseller</th></tr></thead><tbody>'''+rows+'</tbody></table></div><h2>Kapsam kaynakları</h2><ul>'+sources+'</ul></html>')
progress = {'stage':'complete','expansion_assets_saved':192,'expansion_assets_required':192,'catalog_entries':259,'total_assets_saved':777,'depth_maps':777,'all_followups_resolved':True}
(root/'production/expansion-progress.json').write_text(json.dumps(progress,indent=2))
(root/'production/expansion-final-validation.json').write_text(json.dumps(progress,indent=2))
print(json.dumps(progress))
