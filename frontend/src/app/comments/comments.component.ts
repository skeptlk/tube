import { Component, OnInit } from '@angular/core';
import { CommentsService } from '../services';

@Component({
    selector: 'app-comments',
    templateUrl: './comments.component.html',
    styleUrls: ['./comments.component.scss']
})
export class CommentsComponent implements OnInit {

    constructor(private commentsService: CommentsService) { }

    ngOnInit() {
        
    }

}
