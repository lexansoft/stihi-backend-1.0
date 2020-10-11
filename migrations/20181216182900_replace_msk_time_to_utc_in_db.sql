-- articles (time/last_comment_time)
-- comments (time)
-- follows (+time)
-- users (+time/stihi_user_time/last_vote_time)
-- content_votes (time)
-- announces (place_time)
-- invites (place_time)
-- ban_history (time)

ALTER TABLE follows
  ADD COLUMN time TIMESTAMP;

ALTER TABLE users
  ADD COLUMN time TIMESTAMP;

UPDATE articles
  SET time = time - '3 HOUR'::interval
  WHERE time IS NOT NULL;

UPDATE articles
  SET last_comment_time = last_comment_time - '3 HOUR'::interval
  WHERE last_comment_time IS NOT NULL;

UPDATE comments
  SET time = time - '3 HOUR'::interval
  WHERE time IS NOT NULL;

UPDATE users
  SET stihi_user_time = stihi_user_time - '3 HOUR'::interval
  WHERE stihi_user_time IS NOT NULL;

UPDATE users
  SET last_vote_time = last_vote_time - '3 HOUR'::interval
  WHERE last_vote_time IS NOT NULL;

UPDATE content_votes
  SET time = time - '3 HOUR'::interval
  WHERE time IS NOT NULL;

UPDATE announces
  SET place_time = place_time - '3 HOUR'::interval
  WHERE place_time IS NOT NULL;

UPDATE invites
  SET place_time = place_time - '3 HOUR'::interval
  WHERE place_time IS NOT NULL;

UPDATE ban_history
  SET time = time - '3 HOUR'::interval
  WHERE time IS NOT NULL;
