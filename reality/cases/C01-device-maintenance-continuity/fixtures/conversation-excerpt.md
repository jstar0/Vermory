# Authorized device-maintenance conversation excerpt

- The user asked why Gboard lagged and which device data was large.
- Read-only diagnosis found 1,333,470 personal-dictionary rows. The relevant issue was entry cardinality, not merely cache size.
- The user explicitly excluded QQ and WeChat from cleanup conclusions.
- A removed game's resource bundle was identified at about 9.9 GB. The user authorized deleting only that bundle.
- The first deletion command did not affect the target; verification caught the failure.
- A corrected fixed-path command removed the bundle. Final verification showed about 33 MB residual application files, 87 GB free space, and 82 percent data-partition usage.

The device identifier, personal media names, raw paths, account data, and unrelated applications were removed. The deleted game is called Game A in evaluation events.
