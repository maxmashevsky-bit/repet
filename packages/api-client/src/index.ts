export type ApiClientOptions = {
  baseUrl: string;
};

export class ApiClient {
  readonly baseUrl: string;

  constructor(options: ApiClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/$/, "");
  }
}

