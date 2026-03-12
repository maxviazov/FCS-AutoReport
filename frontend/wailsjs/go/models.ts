export namespace domain {
	
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
	export class Settings {
	    inputFolder: string;
	    outputFolder: string;
	    templatePath: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputFolder = source["inputFolder"];
	        this.outputFolder = source["outputFolder"];
	        this.templatePath = source["templatePath"];
	    }
	}

}

