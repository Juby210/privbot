const Discord = require('discord.js');
const client = new Discord.Client();
const prefix = "&";
const { token } = require("./config.json");

client.on('ready', () => {
    console.log(`${client.user.tag} ready`);
});

client.on('message', message => {
    if(message.author.bot) return;
    if (!message.content.startsWith(prefix)) return;

    const args = message.content.slice(prefix.length).trim().split(/ +/g);
    const command = args.shift().toLowerCase();

    if(command == "give") {
        if(!args[0]) return message.channel.send("Give role color hex (#xxxxxx)");
        if (!/^#?([a-fA-F0-9]{6}|[a-fA-F0-9]{3})$/.test(args[0])) return message.channel.send("The color must be in the hex format");
        let color = args[0].toLowerCase();

        let cr;
        message.member.roles.forEach(r => {
            if(r.name.startsWith("color: ")) cr = r;
        });
        if(cr && cr.name == `color: ${color}`) return message.channel.send(`Color added ${color}`);
        if(cr) {
            message.member.removeRole(cr).then(() => {
                if(cr.members.size == 0) cr.delete();
            });
        }

        let role;
        message.guild.roles.forEach(r => {
            if(r.name == `color: ${color}`) role = r;
        });
        message.channel.send(`Color added ${color}`);
        if(role) return message.member.addRole(role).catch(err => message.channel.send("Role adding error"));
        message.guild.createRole({
            name: `color: ${color}`,
            color: color,
            permissions: 0,
            position: message.guild.member(client.user).highestRole.position - 1
        }).then(r => message.member.addRole(r).catch(err => message.channel.send("Role adding error"))).catch(err => {
            message.channel.send("Role creating error because discord api is shit and code is correct");
            message.member.addRole(message.guild.roles.find("name", `color: ${color}`));
        });
    }
    if(command == "clear") {
        let cr;
        message.member.roles.forEach(r => {
            if(r.name.startsWith("color: ")) cr = r;
        });
        if(cr) {
            message.member.removeRole(cr).then(() => {
                if(cr.members.size == 0) cr.delete();
            });
        }
        message.channel.send("Cleared your color");
    }
})

client.login(token);