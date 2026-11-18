export namespace main {
	
	export class TorState {
	    active: boolean;
	    port: number;
	    available: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TorState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active = source["active"];
	        this.port = source["port"];
	        this.available = source["available"];
	    }
	}
	export class WGProfileInfo {
	    ok: boolean;
	    addresses: string[];
	    dns: string[];
	    endpoint: string;
	    peers: number;
	    mtu: number;
	
	    static createFrom(source: any = {}) {
	        return new WGProfileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.addresses = source["addresses"];
	        this.dns = source["dns"];
	        this.endpoint = source["endpoint"];
	        this.peers = source["peers"];
	        this.mtu = source["mtu"];
	    }
	}

}

export namespace settings {
	
	export class Settings {
	    language: string;
	    autostart: boolean;
	    autoconnect: boolean;
	    auto_server: boolean;
	    selected_country: string;
	    selected_server_ip: string;
	    selected_server_name: string;
	    protocol: string;
	    dns_leak_protection: boolean;
	    dns_resolver: string;
	    custom_dns: string;
	    kill_switch: boolean;
	    tor_enabled: boolean;
	    tor_socks_port: number;
	    block_ads: boolean;
	    notifications: boolean;
	    custom_ovpn_name: string;
	    tor_bridges_enabled: boolean;
	    tor_bridges: string;
	    wg_profile: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.language = source["language"];
	        this.autostart = source["autostart"];
	        this.autoconnect = source["autoconnect"];
	        this.auto_server = source["auto_server"];
	        this.selected_country = source["selected_country"];
	        this.selected_server_ip = source["selected_server_ip"];
	        this.selected_server_name = source["selected_server_name"];
	        this.protocol = source["protocol"];
	        this.dns_leak_protection = source["dns_leak_protection"];
	        this.dns_resolver = source["dns_resolver"];
	        this.custom_dns = source["custom_dns"];
	        this.kill_switch = source["kill_switch"];
	        this.tor_enabled = source["tor_enabled"];
	        this.tor_socks_port = source["tor_socks_port"];
	        this.block_ads = source["block_ads"];
	        this.notifications = source["notifications"];
	        this.custom_ovpn_name = source["custom_ovpn_name"];
	        this.tor_bridges_enabled = source["tor_bridges_enabled"];
	        this.tor_bridges = source["tor_bridges"];
	        this.wg_profile = source["wg_profile"];
	    }
	}

}

export namespace vpn {
	
	export class IPInfo {
	    ip: string;
	    country: string;
	    country_code: string;
	    city: string;
	    isp: string;
	
	    static createFrom(source: any = {}) {
	        return new IPInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.country = source["country"];
	        this.country_code = source["country_code"];
	        this.city = source["city"];
	        this.isp = source["isp"];
	    }
	}
	export class Server {
	    HostName: string;
	    IP: string;
	    Score: number;
	    Ping: number;
	    Speed: number;
	    CountryLong: string;
	    CountryShort: string;
	    NumVpnSessions: number;
	    Uptime: string;
	    TotalUsers: string;
	    Operator: string;
	    Message: string;
	    OpenVPN_ConfigData_Base64: string;
	
	    static createFrom(source: any = {}) {
	        return new Server(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.HostName = source["HostName"];
	        this.IP = source["IP"];
	        this.Score = source["Score"];
	        this.Ping = source["Ping"];
	        this.Speed = source["Speed"];
	        this.CountryLong = source["CountryLong"];
	        this.CountryShort = source["CountryShort"];
	        this.NumVpnSessions = source["NumVpnSessions"];
	        this.Uptime = source["Uptime"];
	        this.TotalUsers = source["TotalUsers"];
	        this.Operator = source["Operator"];
	        this.Message = source["Message"];
	        this.OpenVPN_ConfigData_Base64 = source["OpenVPN_ConfigData_Base64"];
	    }
	}

}

export namespace wireguard {
	
	export class State {
	    status: string;
	    profile: string;
	    address: string;
	    endpoint: string;
	    dns: string[];
	    rx: number;
	    tx: number;
	    last_handshake: number;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.profile = source["profile"];
	        this.address = source["address"];
	        this.endpoint = source["endpoint"];
	        this.dns = source["dns"];
	        this.rx = source["rx"];
	        this.tx = source["tx"];
	        this.last_handshake = source["last_handshake"];
	    }
	}

}

