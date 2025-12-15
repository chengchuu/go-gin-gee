// https://www.duolingo.cn/profile/CHENG413

const following = require('./following_20251111.json');
const followers = require('./followers_20251111.json');

const nonMutualFollowing = following.filter(user => {
    return !followers.some(follower => follower.userId === user.userId);
});

const mutualFollowing = following.filter(user => {
    return followers.some(follower => follower.userId === user.userId);
});

console.log(`Number of people I follow: ${following.length}`);
console.log(`Number of followers: ${followers.length}`);
console.log(`Number of non-mutual follows: ${nonMutualFollowing.length}`);
console.log(`Number of mutual follows: ${mutualFollowing.length}`);

nonMutualFollowing.forEach(user => {
    console.log(`Username: ${user.username}, Displayname: ${user.displayName}`);
});
