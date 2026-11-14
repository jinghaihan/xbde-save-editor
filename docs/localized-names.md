# Display names

The nine-language catalogs retain logical game IDs. They were read from the
local version 2.0.0 text tables with Build ID
`7E1DF8E08D60544BBDCA1E333C153C97` using selective archive reads, without a full
RomFS extraction. No save or executable is bundled.

Items follow `ITM_itemlist.itemType/itemID`, then the subtype table's `name`
message reference. Arts, passive skills, gem effects, achievement titles and
Collectopaedia map names likewise follow stored references rather than row
positions. Achievement conditions use `MNU_playaward.comment2`.

Art-manual tiers remain separately identifiable. They use the existing catalog's
validated Intermediate/Advanced/Master metadata, not a guessed localized suffix.
Blank item names retain the existing numeric placeholder.

Skill-tree display translations match the existing English branch labels against
`MNU_PSlink.name -> MNU_personally_name_ms`. They do not infer any save offsets or
character/branch assignments. If multiple matching messages differ in the chosen
language, that label retains the existing English text. This affects nine
French/Spanish branch-label combinations. Regions match the five existing
English labels against `MNU_relate_ms` message IDs 27–31.

`scripts/import_names.py /path/to/XBDE-TEXT-002` regenerates the compact catalogs
from the local ID-preserving `names.json` and `joins.json` export. The export
itself remains in the user's shared directory. No archive paths or keys are
needed at runtime.
