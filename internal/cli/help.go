package cli

const Help = `XBDE Save Editor

inspect <save>
copy <save> <output>
resources <save> <output> [--money N] [--noponstones N]
character <save> <output> <id> [--ap N] [--coins N] [--reserve-exp N]
progression <save> <output> <id> [--level N] [--exp N]
max-ap <save> <output>
arts <save> <character-id>
art <save> <output> <character-id> <art-id> --level N
max-art <save> <output> <character-id> <art-id>
max-arts <save> <output> <character-id>
skills <save> <character-id>
skill-tree <save> <output> <character-id> <tree-index> --learned N
skill-tree <save> <output> <character-id> <tree-index> --sp N
max-skill-tree <save> <output> <character-id> <tree-index>
max-skills <save> <output> <character-id>
max-all-skills <save> <output>
equipment <save> <character-id>
equip <save> <output> <character-id> <Weapon|Head|Torso|Arms|Legs|Feet> <inventory-index>
equipment-gem <save> <output> <character-id> <Weapon|Head|Torso|Arms|Legs|Feet> <socket> <gem-index|none>
--version
gems <save>
gem <save> <output> <gem-index> [--effect N] [--rank N] [--value N]
max-gem <save> <output> <gem-index>
inventory <save> <Collectables|Materials|KeyItems|ArtManuals>
item <save> <output> <kind> <index> --quantity N
add-item <save> <output> <item-id> --quantity N
delete-item <save> <output> <kind> <index>
max-items <save> <output> <kind>
add-gem <save> <output> --effect N --rank N --value N
delete-gem <save> <output> <gem-index>
achievements <save>
unlock-achievement <save> <output> <id>
unlock-all-achievements <save> <output>
repair-achievement-counter <save> <output> <id>
repair-all-achievement-counters <save> <output>
learn-art <save> <output> <character-id> <art-id>
learn-max-arts <save> <output> <character-id>
learn-max-all-arts <save> <output>
skill-links <save> <character-id>
skill-link <save> <output> <character-id> <source-character-id> <slot> <skill-id|none>
affinities <save>
affinity <save> <output> <first-character-id> <second-character-id> --points N
max-affinity <save> <output> <first-character-id> <second-character-id>
max-all-affinity <save> <output>
region-affinities <save>
region-affinity <save> <output> <region-id> --points N
region-affinity <save> <output> <region-id> --stars N
max-region-affinity <save> <output> <region-id>
max-all-region-affinity <save> <output>
equipment-inventory <save> <Weapon|Head|Torso|Arms|Legs|Feet>
add-equipment <save> <output> <item-id>
fill-missing-equipment <save> <output> <slot> <character-id>
equipment-favorite <save> <output> <slot> <index> <true|false>
delete-equipment <save> <output> <slot> <index>
colony6 <save>
max-colony6 <save> <output>
collectopaedia <save>
collectopaedia-plan <save> [--page <map-id>|--entry <entry-id>]
complete-collectopaedia <save> <output> [--page <map-id>|--entry <entry-id>]
unlock-skill-tree <save> <output> <character-id> <tree-index>
unlock-max-skills <save> <output> <character-id>
unlock-max-all-skills <save> <output>`
