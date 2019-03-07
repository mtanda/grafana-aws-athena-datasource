import _ from 'lodash';

export class AwsAthenaDatasourceConfigCtrl {
  /** @ngInject */
  constructor($scope, datasourceSrv) {
    this.current.jsonData.authType = this.current.jsonData.authType || 'credentials';

    this.accessKeyExist = this.current.secureJsonFields.accessKey;
    this.secretKeyExist = this.current.secureJsonFields.secretKey;
    this.datasourceSrv = datasourceSrv;
    this.authTypes = [
      { name: 'Access & secret key', value: 'keys' },
      { name: 'Credentials file', value: 'credentials' },
      { name: 'ARN', value: 'arn' },
    ];
  }

  resetAccessKey() {
    this.accessKeyExist = false;
  }

  resetSecretKey() {
    this.secretKeyExist = false;
  }
}

AwsAthenaDatasourceConfigCtrl.templateUrl = 'partials/config.html';
