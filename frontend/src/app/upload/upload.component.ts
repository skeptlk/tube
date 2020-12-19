import { Component } from '@angular/core';
import { FormControl, FormGroup, Validators } from '@angular/forms';
import { AuthService } from '../services';
import { VideoService } from '../services/videos.service';

@Component({
    selector: 'app-upload',
    templateUrl: './upload.component.html',
    styleUrls: ['./upload.component.scss']
})
export class UploadComponent {

    uploadForm = new FormGroup({
        title: new FormControl('', [Validators.required]),
        description: new FormControl('')
    })

    video: File = null;

    constructor(private vidService: VideoService, private auth: AuthService) { }


    onFileSelected(files: FileList) {
        if (files.length > 0) {
            this.video = files.item(0);
        }
    }

    submit() {
        var fd = new FormData();
        fd.append('video', this.video, this.video.name);
        fd.append('title', this.uploadForm.get('title').value);
        fd.append('description', this.uploadForm.get('description').value);
        fd.append('userID', this.auth.currentUserValue.id?.toString());
        
        this.vidService.upload(fd)
            .subscribe((resp) => console.log(resp) );

    }

}
