export namespace constants {
	
	export enum Platform {
	    windows = 0,
	    darwin = 1,
	    linux = 2,
	}
	export enum Environment {
	    development = 0,
	    production = 1,
	}

}

export namespace dto {
	
	export enum Command {
	    command_stoped = "command_stoped",
	    command_output = "command_output",
	    command_started = "command_started",
	    error = "error",
	}

}

