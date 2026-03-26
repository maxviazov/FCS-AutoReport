export namespace domain {
	
	export class ApprovalResult {
	    jobId: number;
	    invoiceNum: string;
	    clientName: string;
	    clientHP: string;
	    approvalNum: string;
	    status: string;
	    rejectReason: string;
	
	    static createFrom(source: any = {}) {
	        return new ApprovalResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobId = source["jobId"];
	        this.invoiceNum = source["invoiceNum"];
	        this.clientName = source["clientName"];
	        this.clientHP = source["clientHP"];
	        this.approvalNum = source["approvalNum"];
	        this.status = source["status"];
	        this.rejectReason = source["rejectReason"];
	    }
	}
	export class City {
	    id: number;
	    name: string;
	    code: string;
	    aliases: string[];
	
	    static createFrom(source: any = {}) {
	        return new City(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.code = source["code"];
	        this.aliases = source["aliases"];
	    }
	}
	export class Driver {
	    agent_name: string;
	    driver_name: string;
	    car_number: string;
	    phone: string;
	    city_codes: string;
	
	    static createFrom(source: any = {}) {
	        return new Driver(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent_name = source["agent_name"];
	        this.driver_name = source["driver_name"];
	        this.car_number = source["car_number"];
	        this.phone = source["phone"];
	        this.city_codes = source["city_codes"];
	    }
	}
	export class Item {
	    item_code: string;
	    category: string;
	
	    static createFrom(source: any = {}) {
	        return new Item(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.item_code = source["item_code"];
	        this.category = source["category"];
	    }
	}
	export class OutboxJob {
	    id: number;
	    reportPath: string;
	    status: string;
	    error: string;
	    sentAt: string;
	    replyAt: string;
	    subjectHint: string;
	
	    static createFrom(source: any = {}) {
	        return new OutboxJob(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.reportPath = source["reportPath"];
	        this.status = source["status"];
	        this.error = source["error"];
	        this.sentAt = source["sentAt"];
	        this.replyAt = source["replyAt"];
	        this.subjectHint = source["subjectHint"];
	    }
	}
	export class Settings {
	    inputFolder: string;
	    outputFolder: string;
	    templatePath: string;
	    smtpHost: string;
	    smtpPort: number;
	    smtpUser: string;
	    smtpPassword: string;
	    imapHost: string;
	    imapPort: number;
	    imapUser: string;
	    imapPassword: string;
	    autoSend: boolean;
	    watchEnabled: boolean;
	    watchFolder: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputFolder = source["inputFolder"];
	        this.outputFolder = source["outputFolder"];
	        this.templatePath = source["templatePath"];
	        this.smtpHost = source["smtpHost"];
	        this.smtpPort = source["smtpPort"];
	        this.smtpUser = source["smtpUser"];
	        this.smtpPassword = source["smtpPassword"];
	        this.imapHost = source["imapHost"];
	        this.imapPort = source["imapPort"];
	        this.imapUser = source["imapUser"];
	        this.imapPassword = source["imapPassword"];
	        this.autoSend = source["autoSend"];
	        this.watchEnabled = source["watchEnabled"];
	        this.watchFolder = source["watchFolder"];
	    }
	}

}

