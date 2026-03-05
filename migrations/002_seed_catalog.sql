-- +migrate Up

-- Sample quests
INSERT INTO catalog_goals (canonical_key, title, type, description) VALUES
  ('quest.cooks_assistant',      'Cook''s Assistant',        'quest', 'Help the cook at Lumbridge Castle.'),
  ('quest.romeo_and_juliet',     'Romeo & Juliet',           'quest', 'A tale of star-crossed lovers in Varrock.'),
  ('quest.desert_treasure',      'Desert Treasure',          'quest', 'Hunt down four diamonds in the desert.'),
  ('quest.desert_treasure_2',    'Desert Treasure II',       'quest', 'The fallen empire — sequel to Desert Treasure.'),
  ('quest.dragon_slayer',        'Dragon Slayer',            'quest', 'Prove yourself by slaying Elvarg.'),
  ('quest.underground_pass',     'Underground Pass',         'quest', 'Navigate the treacherous Underground Pass.'),
  ('quest.regicide',             'Regicide',                 'quest', 'Assassinate the King of the Elves.'),
  ('quest.monkey_madness',       'Monkey Madness',           'quest', 'Help Gnome King Narnode on Ape Atoll.'),
  ('quest.fremennik_trials',     'The Fremennik Trials',     'quest', 'Earn acceptance among the Fremennik.'),
  ('quest.recipe_for_disaster',  'Recipe for Disaster',      'quest', 'Save the Lumbridge Council from a culinary catastrophe.');

-- Skill requirements for Desert Treasure
INSERT INTO catalog_skill_requirements (catalog_goal_id, skill, level)
SELECT id, 'magic', 50 FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'thieving', 53 FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'firemaking', 50 FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'slayer', 10 FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure';

-- Skill requirements for Dragon Slayer
INSERT INTO catalog_skill_requirements (catalog_goal_id, skill, level)
SELECT id, 'attack', 32 FROM catalog_goals WHERE canonical_key = 'quest.dragon_slayer';

-- Freeform requirements for Desert Treasure
INSERT INTO catalog_requirements (catalog_goal_id, description)
SELECT id, 'The Restless Ghost quest complete' FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'Priest in Peril quest complete' FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'Temple of Ikov quest complete' FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'The Tourist Trap quest complete' FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'Troll Stronghold quest complete' FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'Waterfall Quest complete' FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure';

-- Sample diaries
INSERT INTO catalog_goals (canonical_key, title, type, description) VALUES
  ('diary.lumbridge_easy',       'Lumbridge & Draynor Diary (Easy)',   'diary', 'Easy tasks in the Lumbridge & Draynor area.'),
  ('diary.lumbridge_medium',     'Lumbridge & Draynor Diary (Medium)', 'diary', 'Medium tasks in the Lumbridge & Draynor area.'),
  ('diary.morytania_hard',       'Morytania Diary (Hard)',             'diary', 'Hard tasks in the Morytania area.');

-- Skill requirements for Morytania Hard
INSERT INTO catalog_skill_requirements (catalog_goal_id, skill, level)
SELECT id, 'slayer', 71 FROM catalog_goals WHERE canonical_key = 'diary.morytania_hard'
UNION ALL
SELECT id, 'agility', 61 FROM catalog_goals WHERE canonical_key = 'diary.morytania_hard'
UNION ALL
SELECT id, 'prayer', 70 FROM catalog_goals WHERE canonical_key = 'diary.morytania_hard'
UNION ALL
SELECT id, 'herblore', 53 FROM catalog_goals WHERE canonical_key = 'diary.morytania_hard';

-- +migrate Down
DELETE FROM catalog_skill_requirements;
DELETE FROM catalog_requirements;
DELETE FROM catalog_goals;
