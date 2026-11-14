#!/usr/bin/env python3
"""Build compact display catalogs from a local, ID-preserving text export."""
import argparse
import csv
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('export', type=Path)
    args = parser.parse_args()
    names = json.loads((args.export / 'names.json').read_text())
    joins = json.loads((args.export / 'joins.json').read_text())
    fallback = joins['tree_fallback']
    rows = {r['source_row_id']: r for r in fallback['raw_rows']}
    ambiguous = []
    for candidate in fallback['catalog_name_match_candidates']:
        ids = [str(rows[i]['fields']['name']) for i in candidate['name_matching_source_rows']]
        for language, texts in fallback['texts_by_original_message_id'].items():
            values = {texts[i]['text'] for i in ids}
            # Display-label translation only. Never infer a saved branch offset
            # from the message table's row order; ambiguous translations fall back.
            if len(values) == 1:
                names[language]['tree'][candidate['catalog_key']] = values.pop()
            else:
                ambiguous.append([language, candidate['catalog_key']])
    with (ROOT / 'internal/save/data/collectopaedia.tsv').open() as source:
        assert {r['map_id'] for r in csv.DictReader(source, delimiter='\t')} == set(names['en']['map'])
    region_names = ['Colony 9 Area', 'Colony 6 Area', 'Central Bionis', 'Upper Bionis', 'Hidden Village']
    region_messages = {}
    for i, label in enumerate(region_names, 1):
        matches = [k for k, v in names['en']['region'].items() if v == label]
        assert len(matches) == 1
        region_messages[str(i)] = matches[0]
    manual_tiers = {}
    for item_id, join in joins['joins']['item'].items():
        label = join['catalog_english']
        for tier in ['Intermediate', 'Advanced', 'Master']:
            if label.endswith(f' ({tier})'):
                assert join['arts_manual_metadata']
                manual_tiers[item_id] = tier
    for language, domains in names.items():
        domains['region'] = {k: domains['region'][message] for k, message in region_messages.items()}
        domains['achievement-condition'] = {
            k: v['texts'][language]['comment2']['text']
            for k, v in joins['achievement_descriptions'].items()
        }
        for domain, labels in domains.items():
            domains[domain] = {k: v for k, v in labels.items() if v}
    output = ROOT / 'internal/catalog/data'
    (output / 'names.json').write_text(json.dumps(names, ensure_ascii=False, separators=(',', ':')) + '\n')
    (output / 'manual-tiers.json').write_text(json.dumps(manual_tiers, separators=(',', ':')) + '\n')
    print('Imported languages:', ', '.join(names))
    print('Ambiguous branch-label translations retained in English:', ambiguous)

if __name__ == '__main__':
    main()
